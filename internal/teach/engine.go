package teach

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/cite"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

var (
	_ domain.TeachEngine  = (*Engine)(nil)
	_ domain.ReportReader = (*Engine)(nil)
)

// Options tunes timeouts. Zero values use the defaults.
type Options struct {
	CheckTimeout time.Duration // for the final check at Finish. default 20s
	Log          *slog.Logger
}

// Engine implements domain.TeachEngine.
type Engine struct {
	store   Store
	student domain.LLM // streams the AI student
	checker domain.Checker
	sched   domain.SessionScheduler
	opt     Options

	mu       sync.Mutex
	sessions map[string]*sess // ponytail: in memory, lost on restart. Persist if sessions must survive a deploy.
}

func New(store Store, student domain.LLM, checker domain.Checker, sched domain.SessionScheduler, opt Options) *Engine {
	if opt.CheckTimeout <= 0 {
		opt.CheckTimeout = 20 * time.Second // ponytail: not measured on real models; tune after P0-GATE
	}
	if opt.Log == nil {
		opt.Log = slog.Default()
	}
	return &Engine{store: store, student: student, checker: checker, sched: sched, opt: opt, sessions: map[string]*sess{}}
}

type sess struct {
	id      string
	concept domain.Concept // holds source claims: never sent to the student
	sources []domain.Source

	fin sync.Mutex // serializes Finish so only one final check runs at a time

	mu         sync.Mutex
	turnNo     int
	closed     bool
	taught     []domain.TaughtClaim
	transcript []string
	latex      []string
	used       map[string]bool
	report     *domain.Report // latest Finish result; only a checked one is final
}

func (e *Engine) Start(ctx context.Context, conceptID string) (domain.TeachSession, error) {
	c, srcs, err := e.store.LoadConcept(ctx, conceptID)
	if err != nil {
		return domain.TeachSession{}, err
	}
	ts, err := e.store.StartSession(ctx, conceptID)
	if err != nil {
		return domain.TeachSession{}, err
	}
	e.mu.Lock()
	e.sessions[ts.ID] = &sess{id: ts.ID, concept: c, sources: srcs, used: map[string]bool{}}
	e.mu.Unlock()
	return ts, nil
}

func (e *Engine) get(id string) (*sess, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s, ok := e.sessions[id]
	if !ok {
		return nil, fmt.Errorf("teach: unknown session %q", id)
	}
	return s, nil
}

// Turn saves what the human said and streams the student's next question.
// The checker does not run per turn (DECISIONS D5): it runs once at Finish.
func (e *Engine) Turn(ctx context.Context, sessionID, transcript, latex string) (<-chan domain.Delta, error) {
	began := time.Now()
	s, err := e.get(sessionID)
	if err != nil {
		return nil, err
	}
	transcript, latex = strings.TrimSpace(transcript), strings.TrimSpace(latex)
	if transcript == "" && latex == "" {
		return nil, errors.New("teach: empty turn")
	}
	text := transcript
	if latex != "" {
		text += " [LaTeX: " + latex + "]"
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.New("teach: session already finished")
	}
	n := s.turnNo + 1
	tc, err := e.store.SaveTurn(ctx, domain.TaughtClaim{SessionID: s.id, ConceptID: s.concept.ID, StudentText: text, TurnNo: n}, transcript, latex)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.turnNo = n
	s.taught = append(s.taught, tc)
	s.transcript = append(s.transcript, transcript)
	s.latex = append(s.latex, latex)

	sc := StudentContext{
		ConceptName: s.concept.Name,
		// No checker output per turn: code's generic probes until the Phase 1 matcher.
		Probe: pickProbe(s.concept.Name, n, nil, s.used, s.sources, s.concept.Claims),
	}
	for _, t := range s.taught {
		sc.Taught = append(sc.Taught, t.StudentText)
	}
	s.mu.Unlock()

	sctx, cancel := context.WithCancel(ctx)
	up, err := e.student.Stream(sctx, domain.LLMRequest{
		System:   sc.System(),
		Messages: []domain.Message{{Role: "user", Content: sc.User()}},
	})
	if err != nil {
		cancel()
		return nil, err
	}
	out := make(chan domain.Delta, 16)
	go e.forward(ctx, cancel, s, n, began, up, out)
	return out, nil
}

// forward caps the reply at 2 sentences, logs time to first token and saves
// the reply text.
func (e *Engine) forward(ctx context.Context, cancel context.CancelFunc, s *sess, n int, began time.Time, up <-chan domain.Delta, out chan<- domain.Delta) {
	defer close(out)
	defer cancel()
	sc := sentenceCap{max: 2}
	var reply strings.Builder
	var ttft time.Duration
	send := func(d domain.Delta) bool {
		select {
		case out <- d:
			return true
		case <-ctx.Done():
			return false
		}
	}
loop:
	for d := range up {
		if d.Err != nil {
			e.opt.Log.Warn("student stream failed", "session", s.id, "turn", n, "err", d.Err)
			send(domain.Delta{Err: d.Err})
			break
		}
		t := sc.Feed(d.Text)
		if t != "" {
			if ttft == 0 {
				ttft = time.Since(began)
				e.opt.Log.Info("student ttft", "session", s.id, "turn", n, "ttft_ms", ttft.Milliseconds())
			}
			reply.WriteString(t)
			if !send(domain.Delta{Text: t}) {
				break
			}
		}
		if sc.stopped {
			cancel()
			break loop
		}
	}
	go func() { // drain so the producer never blocks
		for range up {
		}
	}()
	if reply.Len() > 0 {
		if err := e.store.SetAssistantText(context.WithoutCancel(ctx), s.id, n, strings.TrimSpace(reply.String())); err != nil {
			e.opt.Log.Warn("save assistant text failed", "session", s.id, "turn", n, "err", err)
		}
	}
}

// check runs the final check. A panic, error or timeout becomes an error,
// never a partial result.
func (e *Engine) check(ctx context.Context, s *sess, in domain.CheckInput) (res []domain.ClaimResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, err = nil, fmt.Errorf("checker panic: %v", r)
		}
	}()
	cctx, cancel := context.WithTimeout(ctx, e.opt.CheckTimeout)
	defer cancel()
	start := time.Now()
	e.opt.Log.Info("final check started", "session", s.id)
	res, err = e.checker.Check(cctx, in)
	if err == nil {
		e.opt.Log.Info("final check done", "session", s.id, "ms", time.Since(start).Milliseconds())
	}
	return res, err
}

// Finish closes the session to new turns and runs one synchronous final
// check. If the check fails or times out the report is unchecked: no results
// are saved, the scheduler is not called and the session stays open in the
// store, so calling Finish again retries. A checked report is final and later
// calls return it unchanged. The scheduler is keyed by the session id, so
// retries write at most one review event.
func (e *Engine) Finish(ctx context.Context, sessionID string) (domain.Report, error) {
	s, err := e.get(sessionID)
	if err != nil {
		return domain.Report{}, err
	}
	s.fin.Lock()
	defer s.fin.Unlock()
	s.mu.Lock()
	if s.report != nil && s.report.Status == domain.ReportChecked {
		defer s.mu.Unlock()
		return *s.report, nil
	}
	s.closed = true
	in := domain.CheckInput{
		Concept: s.concept, Sources: s.sources, Taught: append([]domain.TaughtClaim(nil), s.taught...),
		Transcript: strings.Join(s.transcript, "\n\n"), LaTeX: strings.TrimSpace(strings.Join(s.latex, "\n")),
	}
	s.mu.Unlock()

	results, err := e.check(ctx, s, in)
	if err != nil {
		if ctx.Err() != nil { // the caller gave up: not a checker outage
			return domain.Report{}, ctx.Err()
		}
		e.opt.Log.Warn("final check failed, report unchecked", "session", s.id, "err", err)
		return e.setReport(s, domain.Report{SessionID: s.id, ConceptID: s.concept.ID, Results: []domain.ClaimResult{}, Status: domain.ReportUnchecked}), nil
	}
	final := finalize(s.concept, s.sources, results)

	if err := e.store.SaveResults(ctx, s.id, final); err != nil {
		return domain.Report{}, err
	}
	ev, err := e.sched.ApplyTeachBackSession(ctx, s.id, s.concept.ID, final)
	if err != nil {
		return domain.Report{}, err
	}
	if err := e.store.FinishSession(ctx, s.id); err != nil {
		e.opt.Log.Warn("mark session finished failed", "session", s.id, "err", err)
	}
	return e.setReport(s, domain.Report{SessionID: s.id, ConceptID: s.concept.ID, Results: final, Schedule: &ev, Status: domain.ReportChecked}), nil
}

func (e *Engine) setReport(s *sess, r domain.Report) domain.Report {
	s.mu.Lock()
	s.report = &r
	s.mu.Unlock()
	return r
}

// GetReport returns the latest report from Finish, checked or unchecked.
func (e *Engine) GetReport(_ context.Context, sessionID string) (domain.Report, error) {
	s, err := e.get(sessionID)
	if err != nil {
		return domain.Report{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.report == nil {
		return domain.Report{}, fmt.Errorf("teach: session %q has no report yet", sessionID)
	}
	return *s.report, nil
}

// finalize applies the rules code owns to the checker's output: drop unknown
// labels, drop any correction without a verified citation (downgrading
// contradicted to partial), and mark required claims the human never
// mentioned as absent.
func finalize(c domain.Concept, sources []domain.Source, in []domain.ClaimResult) []domain.ClaimResult {
	text := map[string]string{}
	for _, s := range sources {
		text[s.ID] = s.Text
	}
	var out []domain.ClaimResult
	seen := map[string]bool{}
	for _, r := range in {
		switch r.Label {
		case domain.LabelSupported, domain.LabelPartial, domain.LabelContradicted, domain.LabelAbsent:
		default:
			continue
		}
		if r.Citation != nil {
			src, ok := text[r.Citation.SourceID]
			if !ok || !cite.Verify(src, r.Citation.Quote) {
				r.Citation = nil
			}
		}
		if r.Citation == nil || (r.Correction != nil && strings.TrimSpace(*r.Correction) == "") {
			// Same rule as the checker: no proof, no correction, no contradiction.
			r.Citation, r.Correction = nil, nil
			if r.Label == domain.LabelContradicted {
				r.Label = domain.LabelPartial
			}
		}
		if r.ClaimID != nil {
			seen[*r.ClaimID] = true
		}
		out = append(out, r)
	}
	for _, cl := range c.Claims {
		if cl.Required && !seen[cl.ID] {
			id := cl.ID
			out = append(out, domain.ClaimResult{ClaimID: &id, Label: domain.LabelAbsent})
		}
	}
	return out
}

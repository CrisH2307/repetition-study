package teach

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var _ domain.TeachEngine = (*Engine)(nil)

// Options tunes timeouts. Zero values use the defaults.
type Options struct {
	CheckTimeout time.Duration // per async check. default 20s
	FinishWait   time.Duration // how long Finish waits for pending checks. default 8s
	Log          *slog.Logger
}

// Engine implements domain.TeachEngine.
type Engine struct {
	store   Store
	student domain.LLM // streams the AI student
	checker domain.Checker
	sched   domain.Scheduler
	opt     Options

	mu       sync.Mutex
	sessions map[string]*sess // ponytail: in memory, lost on restart. Persist if sessions must survive a deploy.
}

func New(store Store, student domain.LLM, checker domain.Checker, sched domain.Scheduler, opt Options) *Engine {
	if opt.CheckTimeout <= 0 {
		opt.CheckTimeout = 20 * time.Second
	}
	if opt.FinishWait <= 0 {
		opt.FinishWait = 8 * time.Second
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

	fin sync.Mutex // serializes Finish so the scheduler runs once
	wg  sync.WaitGroup

	mu         sync.Mutex
	turnNo     int
	closed     bool
	taught     []domain.TaughtClaim
	transcript []string
	latex      []string
	results    []domain.ClaimResult // newest completed check
	resultTurn int
	used       map[string]bool
	report     *domain.Report
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

// Turn saves what the human said, starts the checker in the background and
// streams the student's next question. The reply never waits for the checker.
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

	in := domain.CheckInput{
		Concept: s.concept, Sources: s.sources, Taught: append([]domain.TaughtClaim(nil), s.taught...),
		Transcript: strings.Join(s.transcript, "\n\n"), LaTeX: strings.TrimSpace(strings.Join(s.latex, "\n")),
	}
	sc := StudentContext{
		ConceptName: s.concept.Name,
		Probe:       pickProbe(s.concept.Name, n, s.results, s.used, s.sources, s.concept.Claims),
	}
	for _, t := range s.taught {
		sc.Taught = append(sc.Taught, t.StudentText)
	}
	s.wg.Add(1)
	s.mu.Unlock()

	go e.runCheck(ctx, s, n, in)

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

// runCheck never fails the session: errors, panics and timeouts are logged.
func (e *Engine) runCheck(ctx context.Context, s *sess, n int, in domain.CheckInput) {
	defer s.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			e.opt.Log.Error("checker panic", "session", s.id, "turn", n, "panic", r)
		}
	}()
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), e.opt.CheckTimeout)
	defer cancel()
	start := time.Now()
	res, err := e.checker.Check(cctx, in)
	if err != nil {
		e.opt.Log.Warn("check failed", "session", s.id, "turn", n, "err", err)
		return
	}
	e.opt.Log.Info("check done", "session", s.id, "turn", n, "ms", time.Since(start).Milliseconds())
	s.mu.Lock()
	if n >= s.resultTurn {
		s.results, s.resultTurn = res, n
	}
	s.mu.Unlock()
}

// Finish waits for pending checks up to FinishWait, adds absent results for
// required claims never mentioned, verifies citations, applies the teach-back
// to the schedule once, and returns the report.
func (e *Engine) Finish(ctx context.Context, sessionID string) (domain.Report, error) {
	s, err := e.get(sessionID)
	if err != nil {
		return domain.Report{}, err
	}
	s.fin.Lock()
	defer s.fin.Unlock()
	s.mu.Lock()
	if s.report != nil {
		defer s.mu.Unlock()
		return *s.report, nil
	}
	s.closed = true
	s.mu.Unlock()

	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(e.opt.FinishWait):
		e.opt.Log.Warn("finish: checks still pending, using latest result", "session", s.id)
	case <-ctx.Done():
		return domain.Report{}, ctx.Err()
	}

	s.mu.Lock()
	results := append([]domain.ClaimResult(nil), s.results...)
	s.mu.Unlock()
	final := finalize(s.concept, s.sources, results)

	if err := e.store.SaveResults(ctx, s.id, final); err != nil {
		return domain.Report{}, err
	}
	ev, err := e.sched.ApplyTeachBack(ctx, s.concept.ID, final)
	if err != nil {
		return domain.Report{}, err
	}
	if err := e.store.FinishSession(ctx, s.id); err != nil {
		e.opt.Log.Warn("mark session finished failed", "session", s.id, "err", err)
	}
	rep := domain.Report{SessionID: s.id, ConceptID: s.concept.ID, Results: final, Schedule: &ev}
	s.mu.Lock()
	s.report = &rep
	s.mu.Unlock()
	return rep, nil
}

// finalize applies the rules code owns to the checker's output: drop unknown
// labels, drop any correction without a verified citation, and mark required
// claims the human never mentioned as absent.
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
			if !ok || r.Citation.Quote == "" || !strings.Contains(src, r.Citation.Quote) {
				r.Citation = nil
			}
		}
		if r.Citation == nil {
			r.Correction = nil
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

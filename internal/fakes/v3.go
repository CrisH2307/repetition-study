package fakes

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/cite"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Contract v3 fakes. In memory, deterministic ids (kind-N), every method
// scoped to the user id it is given: another user's id gives ErrNotFound.

var (
	_ domain.AuthStore    = (*AuthStore)(nil)
	_ domain.Auth         = (*Auth)(nil)
	_ domain.CourseStore  = (*CourseStore)(nil)
	_ domain.ReviewStore  = (*ReviewStore)(nil)
	_ domain.CourseNotes  = (*CourseNotes)(nil)
	_ domain.TodayPlanner = TodayPlanner{}
)

// ids hands out kind-1, kind-2, ... per kind.
type ids struct{ n map[string]int }

func (i *ids) next(kind string) string {
	if i.n == nil {
		i.n = map[string]int{}
	}
	i.n[kind]++
	return fmt.Sprintf("%s-%d", kind, i.n[kind])
}

type fakeSession struct {
	user          string
	expires, seen time.Time
}

// AuthStore keeps invites and sessions keyed by the hex of the hash.
type AuthStore struct {
	mu       sync.Mutex
	ids      ids
	invites  map[string]*domain.Invite
	sessions map[string]fakeSession
	Users    map[string]domain.User
	revoked  map[string]bool
}

func (a *AuthStore) init() {
	if a.invites == nil {
		a.invites, a.Users = map[string]*domain.Invite{}, map[string]domain.User{}
		a.sessions = map[string]fakeSession{}
	}
}

func (a *AuthStore) CreateInvite(_ context.Context, name string, h []byte) (domain.Invite, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	inv := &domain.Invite{ID: a.ids.next("invite"), Name: name, CreatedAt: Epoch}
	a.invites[hex.EncodeToString(h)] = inv
	return *inv, nil
}

func (a *AuthStore) RedeemInvite(_ context.Context, h []byte, now time.Time) (domain.User, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	inv, ok := a.invites[hex.EncodeToString(h)]
	if !ok || a.revoked[hex.EncodeToString(h)] {
		return domain.User{}, domain.ErrNotFound
	}
	if inv.UserID == nil {
		u := domain.User{ID: a.ids.next("user"), Name: inv.Name, TimeZone: "America/Toronto"}
		a.Users[u.ID] = u
		inv.UserID, inv.UsedAt = &u.ID, &now
	}
	return a.Users[*inv.UserID], nil
}

func (a *AuthStore) CreateSession(_ context.Context, userID string, h []byte, exp time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	a.sessions[hex.EncodeToString(h)] = fakeSession{user: userID, expires: exp}
	return nil
}

func (a *AuthStore) SessionUser(_ context.Context, h []byte, now time.Time) (domain.User, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	s, ok := a.sessions[hex.EncodeToString(h)]
	if !ok || !now.Before(s.expires) || !s.seen.IsZero() && now.Sub(s.seen) > domain.SessionIdleTTL {
		return domain.User{}, domain.ErrNotFound
	}
	if s.seen.IsZero() || now.Sub(s.seen) > time.Hour {
		s.seen = now
		a.sessions[hex.EncodeToString(h)] = s
	}
	return a.Users[s.user], nil
}

func (a *AuthStore) DeleteUserSessions(_ context.Context, userID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	for k, s := range a.sessions {
		if s.user == userID {
			delete(a.sessions, k)
		}
	}
	return nil
}

func (a *AuthStore) RevokeInvite(_ context.Context, h []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	inv, ok := a.invites[hex.EncodeToString(h)]
	if !ok {
		return domain.ErrNotFound
	}
	if a.revoked == nil {
		a.revoked = map[string]bool{}
	}
	a.revoked[hex.EncodeToString(h)] = true
	if inv.UserID != nil {
		for k, s := range a.sessions {
			if s.user == *inv.UserID {
				delete(a.sessions, k)
			}
		}
	}
	return nil
}

func (a *AuthStore) DeleteSession(_ context.Context, h []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()
	delete(a.sessions, hex.EncodeToString(h))
	return nil
}

// Auth uses plain tokens invite-N and session-N over an AuthStore. Sessions
// last 30 days from Clock (Epoch when nil).
type Auth struct {
	Store *AuthStore
	Clock domain.Clock

	mu  sync.Mutex
	ids ids
}

func (a *Auth) now() time.Time {
	if a.Clock == nil {
		return Epoch
	}
	return a.Clock.Now()
}

func (a *Auth) store() *AuthStore {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Store == nil {
		a.Store = &AuthStore{}
	}
	return a.Store
}

func (a *Auth) token(kind string) string { a.mu.Lock(); defer a.mu.Unlock(); return a.ids.next(kind) }

func (a *Auth) NewInvite(ctx context.Context, name string) (string, error) {
	t := a.token("invite")
	_, err := a.store().CreateInvite(ctx, name, []byte(t))
	return t, err
}

func (a *Auth) Redeem(ctx context.Context, invite string) (domain.User, string, time.Time, error) {
	u, err := a.store().RedeemInvite(ctx, []byte(invite), a.now())
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	t, exp := a.token("session"), a.now().Add(30*24*time.Hour)
	return u, t, exp, a.store().CreateSession(ctx, u.ID, []byte(t), exp)
}

func (a *Auth) UserFor(ctx context.Context, session string) (domain.User, error) {
	return a.store().SessionUser(ctx, []byte(session), a.now())
}

func (a *Auth) SignOut(ctx context.Context, session string) error {
	return a.store().DeleteSession(ctx, []byte(session))
}

func (a *Auth) SignOutEverywhere(ctx context.Context, session string) error {
	u, err := a.UserFor(ctx, session)
	if err != nil {
		return nil
	}
	return a.store().DeleteUserSessions(ctx, u.ID)
}

func (a *Auth) RevokeInvite(ctx context.Context, invite string) error {
	return a.store().RevokeInvite(ctx, []byte(invite))
}

// CourseStore keeps courses and everything under them in memory.
type CourseStore struct {
	mu       sync.Mutex
	ids      ids
	courses  map[string]courseRow
	concepts map[string]domain.Concept // claims kept in claims
	uploads  map[string]uploadRow
	pages    map[string]pageRow
	blocks   map[string]domain.SourceBlock
	pageOf   map[string]string // block id -> page id
	claims   map[string]claimRow
	sources  map[string]domain.Source
}

type courseRow struct {
	user     string
	s        domain.Subject
	archived bool
}

type uploadRow struct {
	u      domain.Upload
	pdf    []byte
	pasted string
}

type pageRow struct {
	p   domain.Page
	img []byte
}

type claimRow struct {
	c       domain.Claim
	deleted bool
}

func (m *CourseStore) init() {
	if m.courses == nil {
		m.courses, m.concepts, m.uploads = map[string]courseRow{}, map[string]domain.Concept{}, map[string]uploadRow{}
		m.pages, m.blocks, m.pageOf = map[string]pageRow{}, map[string]domain.SourceBlock{}, map[string]string{}
		m.claims, m.sources = map[string]claimRow{}, map[string]domain.Source{}
	}
}

// Ownership walks up to the course. Caller holds m.mu.
func (m *CourseStore) course(user, id string) (courseRow, error) {
	m.init()
	c, ok := m.courses[id]
	if !ok || c.user != user || c.archived {
		return courseRow{}, domain.ErrNotFound
	}
	return c, nil
}

func (m *CourseStore) concept(user, id string) (domain.Concept, error) {
	m.init()
	c, ok := m.concepts[id]
	if !ok {
		return domain.Concept{}, domain.ErrNotFound
	}
	_, err := m.course(user, c.SubjectID)
	return c, err
}

func (m *CourseStore) upload(user, id string) (uploadRow, error) {
	m.init()
	u, ok := m.uploads[id]
	if !ok {
		return uploadRow{}, domain.ErrNotFound
	}
	_, err := m.course(user, u.u.SubjectID)
	return u, err
}

func (m *CourseStore) page(user, id string) (pageRow, uploadRow, error) {
	m.init()
	p, ok := m.pages[id]
	if !ok {
		return pageRow{}, uploadRow{}, domain.ErrNotFound
	}
	u, err := m.upload(user, p.p.UploadID)
	return p, u, err
}

func (m *CourseStore) claim(user, id string) (claimRow, error) {
	m.init()
	c, ok := m.claims[id]
	if !ok || c.deleted {
		return claimRow{}, domain.ErrNotFound
	}
	_, err := m.concept(user, c.c.ConceptID)
	return c, err
}

func (m *CourseStore) ListCourses(_ context.Context, user string) ([]domain.Subject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	out := []domain.Subject{}
	for _, c := range m.courses {
		if c.user == user && !c.archived {
			out = append(out, c.s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *CourseStore) GetCourse(_ context.Context, user, id string) (domain.Subject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.course(user, id)
	return c.s, err
}

func (m *CourseStore) CreateCourse(_ context.Context, user string, s domain.Subject) (domain.Subject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	s.ID = m.ids.next("course")
	m.courses[s.ID] = courseRow{user: user, s: s}
	return s, nil
}

func (m *CourseStore) UpdateCourse(_ context.Context, user string, s domain.Subject) (domain.Subject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.course(user, s.ID)
	if err != nil {
		return domain.Subject{}, err
	}
	c.s = s
	m.courses[s.ID] = c
	return s, nil
}

func (m *CourseStore) ArchiveCourse(_ context.Context, user, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.course(user, id)
	if err != nil {
		return err
	}
	c.archived = true
	m.courses[id] = c
	return nil
}

func (m *CourseStore) CreateConcept(_ context.Context, user, subjectID, name string) (domain.Concept, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.course(user, subjectID); err != nil {
		return domain.Concept{}, err
	}
	c := domain.Concept{ID: m.ids.next("concept"), SubjectID: subjectID, Name: name, Claims: []domain.Claim{}}
	m.concepts[c.ID] = c
	return c, nil
}

func (m *CourseStore) GetConcept(_ context.Context, user, id string) (domain.Concept, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.concept(user, id)
	if err != nil {
		return domain.Concept{}, err
	}
	c.Claims = m.liveClaims(id)
	return c, nil
}

func (m *CourseStore) liveClaims(conceptID string) []domain.Claim {
	out := []domain.Claim{}
	for _, c := range m.claims {
		if c.c.ConceptID == conceptID && !c.deleted {
			out = append(out, c.c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *CourseStore) CreateUpload(_ context.Context, user string, u domain.Upload, pdf []byte, pasted string) (domain.Upload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.course(user, u.SubjectID); err != nil {
		return domain.Upload{}, err
	}
	if (len(pdf) == 0) == (pasted == "") {
		return domain.Upload{}, &domain.UserError{Msg: "Add a PDF or paste some text, not both."}
	}
	if len(pdf) > 20<<20 {
		return domain.Upload{}, &domain.UserError{Msg: "The file is larger than 20 MB."}
	}
	u.ID, u.Status, u.CreatedAt = m.ids.next("upload"), domain.UploadReading, Epoch
	m.uploads[u.ID] = uploadRow{u: u, pdf: pdf, pasted: pasted}
	return u, nil
}

// withCounts fills PagesTotal and PagesReady. Caller holds m.mu.
func (m *CourseStore) withCounts(u domain.Upload) domain.Upload {
	u.PagesTotal, u.PagesReady = 0, 0
	for _, p := range m.pages {
		if p.p.UploadID == u.ID {
			u.PagesTotal++
			if p.p.Status == domain.PageReady {
				u.PagesReady++
			}
		}
	}
	return u
}

func (m *CourseStore) GetUpload(_ context.Context, user, id string) (domain.Upload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, err := m.upload(user, id)
	return m.withCounts(u.u), err
}

func (m *CourseStore) ListUploads(_ context.Context, user, subjectID string) ([]domain.Upload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.course(user, subjectID); err != nil {
		return nil, err
	}
	out := []domain.Upload{}
	for _, u := range m.uploads {
		if u.u.SubjectID == subjectID {
			out = append(out, m.withCounts(u.u))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *CourseStore) UploadFile(_ context.Context, user, id string) ([]byte, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, err := m.upload(user, id)
	return u.pdf, u.pasted, err
}

func (m *CourseStore) SetUploadStatus(_ context.Context, user, id string, st domain.UploadStatus, msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, err := m.upload(user, id)
	if err != nil {
		return err
	}
	u.u.Status, u.u.Error = st, msg
	m.uploads[id] = u
	return nil
}

func (m *CourseStore) SavePages(_ context.Context, user, uploadID string, pages []domain.Page, images [][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.upload(user, uploadID); err != nil {
		return err
	}
	for i, p := range pages {
		p.ID, p.UploadID, p.Status, p.UpdatedAt = m.ids.next("page"), uploadID, domain.PageNeedsChecking, Epoch
		var img []byte
		if i < len(images) {
			img = images[i]
		}
		p.HasImage = img != nil
		m.pages[p.ID] = pageRow{p: p, img: img}
	}
	return nil
}

func (m *CourseStore) ListPages(_ context.Context, user, uploadID string) ([]domain.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.upload(user, uploadID); err != nil {
		return nil, err
	}
	out := []domain.Page{}
	for _, p := range m.pages {
		if p.p.UploadID == uploadID {
			out = append(out, p.p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

func (m *CourseStore) GetPage(_ context.Context, user, id string) (domain.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, _, err := m.page(user, id)
	return p.p, err
}

func (m *CourseStore) PageImage(_ context.Context, user, id string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, _, err := m.page(user, id)
	if err == nil && p.img == nil {
		err = domain.ErrNotFound
	}
	return p.img, err
}

func (m *CourseStore) EditPage(_ context.Context, user, id, text string) (domain.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, _, err := m.page(user, id)
	if err != nil {
		return domain.Page{}, err
	}
	p.p.EditedText = text
	m.pages[id] = p
	return p.p, nil
}

func (m *CourseStore) SetPageReady(_ context.Context, user, id string, blocks []domain.SourceBlock) (domain.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, u, err := m.page(user, id)
	if err != nil {
		return domain.Page{}, err
	}
	if u.u.SourceID == nil {
		src := domain.Source{ID: m.ids.next("source"), Kind: domain.SourceCourse, Title: u.u.Filename}
		m.sources[src.ID] = src
		u.u.SourceID = &src.ID
		m.uploads[u.u.ID] = u
	}
	m.dropBlocks(id)
	for _, b := range blocks {
		b.ID, b.SourceID = m.ids.next("block"), *u.u.SourceID
		m.blocks[b.ID], m.pageOf[b.ID] = b, id
	}
	p.p.Status = domain.PageReady
	m.pages[id] = p
	return p.p, nil
}

func (m *CourseStore) dropBlocks(pageID string) {
	for b, p := range m.pageOf {
		if p == pageID {
			delete(m.blocks, b)
			delete(m.pageOf, b)
		}
	}
}

func (m *CourseStore) SetPageNotReady(_ context.Context, user, id string) (domain.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, _, err := m.page(user, id)
	if err != nil {
		return domain.Page{}, err
	}
	for _, c := range m.claims {
		if !c.deleted && c.c.Anchor != nil && m.pageOf[c.c.Anchor.BlockID] == id {
			return domain.Page{}, &domain.UserError{Msg: "A claim quotes this page. Remove that claim first."}
		}
	}
	m.dropBlocks(id)
	p.p.Status = domain.PageNeedsChecking
	m.pages[id] = p
	return p.p, nil
}

func (m *CourseStore) ReadyBlocks(_ context.Context, user, subjectID string) ([]domain.SourceBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.course(user, subjectID); err != nil {
		return nil, err
	}
	return m.readyBlocks(subjectID), nil
}

func (m *CourseStore) readyBlocks(subjectID string) []domain.SourceBlock {
	out := []domain.SourceBlock{}
	for b, pid := range m.pageOf {
		if m.uploads[m.pages[pid].p.UploadID].u.SubjectID == subjectID {
			out = append(out, m.blocks[b])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// CourseSources joins each source's ready blocks into its Text.
func (m *CourseStore) CourseSources(_ context.Context, user, subjectID string) ([]domain.Source, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.course(user, subjectID); err != nil {
		return nil, err
	}
	text := map[string]string{}
	for _, b := range m.readyBlocks(subjectID) {
		text[b.SourceID] += b.TextMD + "\n\n"
	}
	out := []domain.Source{}
	for id, t := range text {
		s := m.sources[id]
		s.Text = t
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *CourseStore) ListClaims(_ context.Context, user, conceptID string) ([]domain.Claim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.concept(user, conceptID); err != nil {
		return nil, err
	}
	return m.liveClaims(conceptID), nil
}

// fill sets the checklist fields from in. Caller holds m.mu.
func (m *CourseStore) fill(c domain.Claim, user string, in domain.ClaimInput) domain.Claim {
	kind, anchor, author := in.Kind, in.Anchor, user
	c.Text, c.Required, c.Kind, c.Anchor, c.AuthorUserID = in.Text, in.Required, &kind, &anchor, &author
	c.SourceID = m.blocks[in.Anchor.BlockID].SourceID
	return c
}

func (m *CourseStore) CreateClaim(_ context.Context, user, conceptID string, in domain.ClaimInput) (domain.Claim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.concept(user, conceptID); err != nil {
		return domain.Claim{}, err
	}
	c := m.fill(domain.Claim{ID: m.ids.next("claim"), ConceptID: conceptID}, user, in)
	m.claims[c.ID] = claimRow{c: c}
	return c, nil
}

func (m *CourseStore) UpdateClaim(_ context.Context, user, id string, in domain.ClaimInput) (domain.Claim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.claim(user, id)
	if err != nil {
		return domain.Claim{}, err
	}
	c.c = m.fill(c.c, user, in)
	m.claims[id] = c
	return c.c, nil
}

func (m *CourseStore) DeleteClaim(_ context.Context, user, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, err := m.claim(user, id)
	if err != nil {
		return err
	}
	c.deleted = true
	m.claims[id] = c
	return nil
}

// ReviewStore keeps self-grades and disputes; CourseView and StudyState
// return what tests put in Views and States (keyed by user id, then subject id).
type ReviewStore struct {
	Views    map[string]map[string]domain.CourseView
	States   map[string]domain.StudyState
	Sessions map[string]string // session id -> owner user id

	mu         sync.Mutex
	ids        ids
	SelfGrades map[string][]domain.SelfGrade
	Disputes   []domain.Dispute
}

func (r *ReviewStore) owns(user, session string) error {
	if r.Sessions[session] != user {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ReviewStore) SaveSelfGrades(_ context.Context, user, session string, g []domain.SelfGrade) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.owns(user, session); err != nil {
		return err
	}
	if r.SelfGrades == nil {
		r.SelfGrades = map[string][]domain.SelfGrade{}
	}
	r.SelfGrades[session] = g
	return nil
}

func (r *ReviewStore) CreateDispute(_ context.Context, user string, d domain.Dispute) (domain.Dispute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.owns(user, d.SessionID); err != nil {
		return domain.Dispute{}, err
	}
	d.ID, d.UserID, d.Status, d.CreatedAt = r.ids.next("dispute"), user, domain.DisputeOpen, Epoch
	r.Disputes = append(r.Disputes, d)
	return d, nil
}

func (r *ReviewStore) AllDisputes(context.Context) ([]domain.Dispute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Dispute{}, r.Disputes...), nil
}

func (r *ReviewStore) CourseView(_ context.Context, user, subjectID string) (domain.CourseView, error) {
	v, ok := r.Views[user][subjectID]
	if !ok {
		return domain.CourseView{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *ReviewStore) StudyState(_ context.Context, user string, _ time.Time) (domain.StudyState, error) {
	s, ok := r.States[user]
	if !ok {
		return domain.StudyState{User: domain.User{ID: user}, Courses: []domain.CourseState{}}, nil
	}
	return s, nil
}

func (r *ReviewStore) SessionOwner(_ context.Context, user, session string) error {
	return r.owns(user, session)
}

// CourseNotes is the fake course logic over a CourseStore. Pasted text is one
// page; a PDF becomes one page reading "fake page". MarkReady makes one block
// of the edited text; anchors are checked with internal/cite.
type CourseNotes struct{ Store *CourseStore }

func (n CourseNotes) ReadUpload(ctx context.Context, user, id string) error {
	pdf, pasted, err := n.Store.UploadFile(ctx, user, id)
	if err != nil {
		return err
	}
	text := pasted
	if len(pdf) > 0 {
		text = "fake page"
	}
	if err := n.Store.SavePages(ctx, user, id, []domain.Page{{Number: 1, RawText: text, EditedText: text}}, nil); err != nil {
		return err
	}
	return n.Store.SetUploadStatus(ctx, user, id, domain.UploadNeedsChecking, "")
}

func (n CourseNotes) EditPage(ctx context.Context, user, id, text string) (domain.Page, error) {
	return n.Store.EditPage(ctx, user, id, text)
}

func (n CourseNotes) MarkReady(ctx context.Context, user, id string) (domain.Page, error) {
	p, err := n.Store.GetPage(ctx, user, id)
	if err != nil {
		return domain.Page{}, err
	}
	b := domain.SourceBlock{Page: &p.Number, Kind: domain.BlockText, TextMD: p.EditedText, TextNorm: cite.Normalize(p.EditedText), VerifiedBy: user}
	return n.Store.SetPageReady(ctx, user, id, []domain.SourceBlock{b})
}

func (n CourseNotes) MarkNotReady(ctx context.Context, user, id string) (domain.Page, error) {
	return n.Store.SetPageNotReady(ctx, user, id)
}

func (n CourseNotes) check(ctx context.Context, user, conceptID string, in domain.ClaimInput) error {
	c, err := n.Store.GetConcept(ctx, user, conceptID)
	if err != nil {
		return err
	}
	blocks, err := n.Store.ReadyBlocks(ctx, user, c.SubjectID)
	if err != nil {
		return err
	}
	for _, b := range blocks {
		if b.ID == in.Anchor.BlockID && cite.Verify(b.TextMD, in.Anchor.Quote) {
			return nil
		}
	}
	return &domain.UserError{Msg: "That quote is not on a Ready page of this course."}
}

func (n CourseNotes) AddClaim(ctx context.Context, user, conceptID string, in domain.ClaimInput) (domain.Claim, error) {
	if err := n.check(ctx, user, conceptID, in); err != nil {
		return domain.Claim{}, err
	}
	return n.Store.CreateClaim(ctx, user, conceptID, in)
}

func (n CourseNotes) EditClaim(ctx context.Context, user, id string, in domain.ClaimInput) (domain.Claim, error) {
	n.Store.mu.Lock()
	c, err := n.Store.claim(user, id)
	n.Store.mu.Unlock()
	if err != nil {
		return domain.Claim{}, err
	}
	if err := n.check(ctx, user, c.c.ConceptID, in); err != nil {
		return domain.Claim{}, err
	}
	return n.Store.UpdateClaim(ctx, user, id, in)
}

func (n CourseNotes) DeleteClaim(ctx context.Context, user, id string) error {
	return n.Store.DeleteClaim(ctx, user, id)
}

// TodayPlanner: concepts due by the end of now's UTC day, or never taught.
// Never taught is teach (15 min), else recall (5 min). Budget is weekly_hours,
// used is 0. Times are UTC; the real planner uses the user's zone.
type TodayPlanner struct{}

func (TodayPlanner) Today(st domain.StudyState, now time.Time) (domain.Today, error) {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	t := domain.Today{Date: day, Courses: []domain.TodayCourse{}, StudyTime: []domain.CourseHours{}}
	for _, c := range st.Courses {
		tc := domain.TodayCourse{SubjectID: c.Subject.ID, Name: c.Subject.Name, Items: []domain.TodayItem{}}
		for _, k := range c.Concepts {
			if k.DueAt != nil && !k.DueAt.Before(day.AddDate(0, 0, 1)) {
				continue
			}
			it := domain.TodayItem{ConceptID: k.ConceptID, ConceptName: k.Name, SubjectID: c.Subject.ID,
				Kind: domain.TodayRecall, Minutes: 5, DueAt: day, Reason: "due for review"}
			if k.DueAt != nil {
				it.DueAt = *k.DueAt
			}
			if !k.Taught {
				it.Kind, it.Minutes, it.Reason = domain.TodayTeach, 15, "not taught yet"
			}
			tc.Items = append(tc.Items, it)
			tc.Minutes += it.Minutes
		}
		t.Courses = append(t.Courses, tc)
		t.StudyTime = append(t.StudyTime, domain.CourseHours{SubjectID: c.Subject.ID, Name: c.Subject.Name, HoursBudget: c.Subject.WeeklyHours})
	}
	return t, nil
}

func (p TodayPlanner) Week(st domain.StudyState, weekStart, now time.Time) (domain.WeekPlan, error) {
	t, _ := p.Today(st, now)
	w := domain.WeekPlan{WeekStart: weekStart, Courses: t.StudyTime, Notes: []string{}}
	for i := range 7 {
		d := domain.WeekDay{Date: weekStart.AddDate(0, 0, i), Items: []domain.TodayItem{}}
		if d.Date.Equal(t.Date) {
			for _, c := range t.Courses {
				d.Items = append(d.Items, c.Items...)
			}
		}
		w.Days = append(w.Days, d)
	}
	return w, nil
}

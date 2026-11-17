package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/internal/store"
)

// A revoked share, another student's share id and a signed-out caller all fail.
func TestIsolationV5(t *testing.T) {
	e := newEnv(t, &fakes.LLM{Reply: "Why?"}, &fakes.LLM{JSONReply: `{"results":[]}`}, "")
	a, b := e.newStudent("Ana"), e.newStudent("Bo")
	ac := a.makeCourse("Ana's course", "Anamarker")
	bc := b.makeCourse("Bo's course", "Bomarker")

	var ns domain.NewShare
	a.call("POST", "/v1/courses/"+ac.ID+"/share", "", 201, &ns)
	if ns.Token == "" {
		t.Fatal("no token")
	}
	// Bo cannot share Ana's course or revoke Ana's link.
	b.call("POST", "/v1/courses/"+ac.ID+"/share", "", 404, nil)
	b.call("DELETE", "/v1/shares/"+ns.ID, "", 404, nil)
	// Only instructors share notes.
	a.call("POST", "/v1/courses/"+ac.ID+"/share", `{"include_sources":true}`, 400, nil)
	// Bo previews and imports the live link into his own course; never into Ana's.
	var p domain.PackPreview
	b.call("GET", "/v1/shared/"+ns.Token, "", 200, &p)
	if p.OwnerName != "Ana" {
		t.Errorf("owner = %q", p.OwnerName)
	}
	b.call("POST", "/v1/shared/"+ns.Token+"/import", fmt.Sprintf(`{"subject_id":%q}`, ac.ID), 404, nil)
	b.call("POST", "/v1/shared/"+ns.Token+"/import", fmt.Sprintf(`{"subject_id":%q}`, bc.ID), 200, nil)
	// No public reads.
	if r := do(NewHandler(e.d), "GET", "/v1/shared/"+ns.Token, ""); r.Code != 401 {
		t.Errorf("signed out preview: %d", r.Code)
	}
	// Revoked: 404 for everyone, and the owner cannot revoke twice.
	a.call("DELETE", "/v1/shares/"+ns.ID, "", 204, nil)
	b.call("GET", "/v1/shared/"+ns.Token, "", 404, nil)
	b.call("POST", "/v1/shared/"+ns.Token+"/import", fmt.Sprintf(`{"subject_id":%q}`, bc.ID), 404, nil)
	// An instructor may share notes, and /v1/me says so.
	var me domain.User
	a.call("GET", "/v1/me", "", 200, &me)
	if me.IsInstructor {
		t.Fatal("Ana is not an instructor yet")
	}
	if err := store.New(e.pool).SetInstructor(context.Background(), me.ID, true); err != nil {
		t.Fatal(err)
	}
	a.call("GET", "/v1/me", "", 200, &me)
	if !me.IsInstructor {
		t.Error("/v1/me must report is_instructor")
	}
	a.call("POST", "/v1/courses/"+ac.ID+"/share", `{"include_sources":true}`, 201, nil)
}

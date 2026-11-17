package store_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

func TestShareStore(t *testing.T) {
	s := store.New(openDB(t)())
	ann, bob := newUser(t, s, "ann"), newUser(t, s, "bob")
	c, err := s.CreateCourse(bg, ann.ID, domain.Subject{Name: "Info", WeeklyHours: 1, Level: "intermediate", Weight: 1})
	mustOK(t, err)
	h := sha256.Sum256([]byte("tok"))

	// only the owner can share the course
	_, err = s.CreateShare(bg, bob.ID, c.ID, h[:], false)
	isNF(t, "create on ann's course", err)
	sh, err := s.CreateShare(bg, ann.ID, c.ID, h[:], false)
	mustOK(t, err)
	if sh.ID == "" || sh.OwnerID != ann.ID || sh.RevokedAt != nil {
		t.Fatalf("share: %+v", sh)
	}
	got, err := s.ShareByToken(bg, h[:])
	mustOK(t, err)
	if got.ID != sh.ID || got.SubjectID != c.ID || got.OwnerID != ann.ID {
		t.Fatalf("by token: %+v", got)
	}
	_, err = s.ShareByToken(bg, []byte("wrong"))
	isNF(t, "unknown hash", err)

	// revoke: not by bob, once by ann, then 404
	isNF(t, "bob revokes", s.RevokeShare(bg, bob.ID, sh.ID, time.Now()))
	isNF(t, "made-up id", s.RevokeShare(bg, ann.ID, "nope", time.Now()))
	mustOK(t, s.RevokeShare(bg, ann.ID, sh.ID, time.Now()))
	mustOK(t, s.RevokeShare(bg, ann.ID, sh.ID, time.Now().Add(time.Hour)))
	_, err = s.ShareByToken(bg, h[:])
	isNF(t, "revoked", err)

	// instructor flag
	u, err := s.UserByID(bg, ann.ID)
	mustOK(t, err)
	if u.IsInstructor || u.Name != "ann" {
		t.Fatalf("user: %+v", u)
	}
	mustOK(t, s.SetInstructor(bg, ann.ID, true))
	if u, _ = s.UserByID(bg, ann.ID); !u.IsInstructor {
		t.Fatal("flag not set")
	}
	mustOK(t, s.SetInstructor(bg, ann.ID, false))
	if u, _ = s.UserByID(bg, ann.ID); u.IsInstructor {
		t.Fatal("flag not cleared")
	}
	isNF(t, "no such user", s.SetInstructor(bg, "00000000-0000-0000-0000-0000000000ff", true))

	// archived course hides its links
	h2 := sha256.Sum256([]byte("tok2"))
	_, err = s.CreateShare(bg, ann.ID, c.ID, h2[:], false)
	mustOK(t, err)
	mustOK(t, s.ArchiveCourse(bg, ann.ID, c.ID))
	_, err = s.ShareByToken(bg, h2[:])
	isNF(t, "archived course", err)
}

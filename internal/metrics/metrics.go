// Package metrics computes pilot numbers (D36) from existing rows. No tracking.
package metrics

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"io"
	"strconv"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Week is one CSV row.
type Week struct {
	WeekStart          time.Time
	ActiveUsers        int
	Sessions           int
	SessionsPerUser    float64 // 0 when nobody was active
	DisputesPerChecked float64 // disputes / checked sessions, 0 when none checked
	UncheckedRate      float64 // unchecked / sessions, 0 when no sessions
}

// Report is the whole export. Week4Retention is -1 when no user is old enough.
type Report struct {
	Weeks          []Week
	Week4Retention float64
	Week4Eligible  int
	Upheld         int // upheld disputes, filled by the caller (EvalCaseStore)
}

// Monday returns 00:00 UTC of the Monday on or before t.
func Monday(t time.Time) time.Time {
	t = t.UTC().Truncate(24 * time.Hour)
	return t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
}

// Compute reads weeks in [from, to) and users, as of now.
func Compute(ctx context.Context, s domain.MetricsStore, from, to, now time.Time) (Report, error) {
	ws, err := s.WeekCounts(ctx, from, to)
	if err != nil {
		return Report{}, err
	}
	us, err := s.UserWeeks(ctx)
	if err != nil {
		return Report{}, err
	}
	r := Report{Week4Retention: -1}
	for _, w := range ws {
		row := Week{WeekStart: w.WeekStart, ActiveUsers: w.ActiveUsers, Sessions: w.Sessions}
		if w.ActiveUsers > 0 {
			row.SessionsPerUser = float64(w.Sessions) / float64(w.ActiveUsers)
		}
		if w.CheckedSessions > 0 {
			row.DisputesPerChecked = float64(w.Disputes) / float64(w.CheckedSessions)
		}
		if w.Sessions > 0 {
			row.UncheckedRate = float64(w.UncheckedSessions) / float64(w.Sessions)
		}
		r.Weeks = append(r.Weeks, row)
	}
	kept := 0
	for _, u := range us {
		w4 := Monday(u.JoinedAt).AddDate(0, 0, 28)
		if now.Before(w4.AddDate(0, 0, 7)) { // week 4 not over yet
			continue
		}
		r.Week4Eligible++
		for _, a := range u.ActiveWeeks {
			if Monday(a).Equal(w4) {
				kept++
				break
			}
		}
	}
	if r.Week4Eligible > 0 {
		r.Week4Retention = float64(kept) / float64(r.Week4Eligible)
	}
	return r, nil
}

// HashID gives a stable short id that hides the real one. Salt keeps ids from
// being guessed back; use the same salt to compare two exports.
func HashID(salt, id string) string {
	h := sha256.Sum256([]byte(salt + ":" + id))
	return hex.EncodeToString(h[:6])
}

func f(x float64) string { return strconv.FormatFloat(x, 'f', 3, 64) }

// WriteCSV writes one row per week, then summary rows, then hashed users.
func WriteCSV(out io.Writer, r Report, us []domain.UserWeeks, salt string) error {
	w := csv.NewWriter(out)
	w.Write([]string{"week_start", "active_users", "sessions", "sessions_per_active_user", "disputes_per_checked_session", "unchecked_rate"})
	for _, k := range r.Weeks {
		w.Write([]string{k.WeekStart.UTC().Format("2006-01-02"), strconv.Itoa(k.ActiveUsers), strconv.Itoa(k.Sessions), f(k.SessionsPerUser), f(k.DisputesPerChecked), f(k.UncheckedRate)})
	}
	w.Write(nil)
	w.Write([]string{"summary", "value"})
	ret := "n/a"
	if r.Week4Retention >= 0 {
		ret = f(r.Week4Retention)
	}
	w.Write([]string{"week4_retention", ret})
	w.Write([]string{"week4_eligible_users", strconv.Itoa(r.Week4Eligible)})
	w.Write([]string{"upheld_disputes", strconv.Itoa(r.Upheld)})
	w.Write(nil)
	w.Write([]string{"user_hash", "joined_week", "active_weeks"})
	for _, u := range us {
		w.Write([]string{HashID(salt, u.UserID), Monday(u.JoinedAt).Format("2006-01-02"), strconv.Itoa(len(u.ActiveWeeks))})
	}
	w.Flush()
	return w.Error()
}

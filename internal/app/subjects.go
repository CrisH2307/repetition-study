package app

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// pgSubjects lists and creates the demo user's subjects.
type pgSubjects struct{ db DB }

func (p pgSubjects) List(ctx context.Context) ([]domain.Subject, error) {
	rows, err := p.db.Query(ctx, `SELECT id, name, weekly_hours::float8, level, exam_date, weight::float8
		FROM subjects WHERE user_id = $1 ORDER BY created_at, id`, domain.DemoUserID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (s domain.Subject, err error) {
		err = r.Scan(&s.ID, &s.Name, &s.WeeklyHours, &s.Level, &s.ExamDate, &s.Weight)
		return
	})
}

func (p pgSubjects) Create(ctx context.Context, s domain.Subject) (domain.Subject, error) {
	err := p.db.QueryRow(ctx, `INSERT INTO subjects (user_id, name, weekly_hours, level, exam_date, weight)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		domain.DemoUserID, s.Name, s.WeeklyHours, s.Level, s.ExamDate, s.Weight).Scan(&s.ID)
	return s, err
}

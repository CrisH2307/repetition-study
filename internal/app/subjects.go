package app

import (
	"context"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// courseSubjects serves the Phase 0 /v1/subjects routes from the course store,
// so they list and create the signed-in user's courses.
type courseSubjects struct{ domain.CourseStore }

func (c courseSubjects) List(ctx context.Context, userID string) ([]domain.Subject, error) {
	return c.ListCourses(ctx, userID)
}

func (c courseSubjects) Create(ctx context.Context, userID string, s domain.Subject) (domain.Subject, error) {
	return c.CreateCourse(ctx, userID, s)
}

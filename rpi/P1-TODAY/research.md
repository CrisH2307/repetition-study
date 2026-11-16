# P1-TODAY research
- Contract: `domain.TodayPlanner`, `StudyState`, `Today`, `WeekPlan` in internal/domain/v3.go. Fake in internal/fakes/v3.go uses UTC; this replaces it.
- User time zone is `StudyState.User.TimeZone` (IANA; empty means UTC; unknown name is an error).
- Days are civil dates in that zone, so DST weeks have 23 or 25 hour days and still split correctly.
- Name `planner.Today{}` (struct) satisfies the interface. Old `Allocate` code is untouched.
- Rules chosen (unverified with Cris): used = 15 min per teach_back event + 5 per card_review in the local week; budget only rises, in 0.5 h steps, to fit the work due that week.

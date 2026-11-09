# U4 research

- Module: `github.com/open-spaced-repetition/go-fsrs/v4` v4.0.0 (latest stable; go.mod already pins it, not touched). FSRS-6 default weights (`DefaultWeights`, W[20] decay).
- API (read from the module source): `fsrs.NewFSRS(Parameters)`, `Next(card, now, Rating) (SchedulingInfo, error)`, `Repeat` (all four ratings), `NewCard(now)`. `Card` fields: Due, Stability, Difficulty, ScheduledDays, Reps, Lapses, State, LastReview. Errors on LastReview after now.
- Default params use minute learning steps (Again 1m, Hard 6m, Good 10m on a new card). Useless for a weekly plan. We keep default weights and retention 0.9 but set `EnableShortTerm=false`: new card gives Again 1d, Hard 2d, Good 3d. Unverified against the FSRS authors' intent; it is a one-line switch in `newFSRS`.
- `go build` with `pgxpool` wants `puddle` in go.mod, which is off limits. `srs.DB` is a small interface that `*pgxpool.Pool` and `*pgx.Conn` both satisfy.
- Prompt says cards have their own FSRS state; U0 contract says concept is the FSRS unit and `cards` has no schedule columns. Followed the contract: `ReviewCard` grades the concept, event carries `card_id`.
- Interface `Scheduler` has no event id parameter and `DueItem` has no reason field. Contract is frozen, so: event id rides on context (`srs.WithEventID`), reasons come from `DueWithReasons` (`Due` wraps it). Cris may want to add both to the contract.
- `cards` has no status column, so "open gap card" means "a card exists for that claim".

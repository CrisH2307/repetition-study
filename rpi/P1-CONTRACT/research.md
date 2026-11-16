# P1-CONTRACT research

- Phase 0 contract: `internal/domain` (types.go, interfaces.go), `api/openapi.yaml` 0.2.0, migrations 0001 to 0003.
- `subjects.user_id` already exists (default demo user). `sources` had no link to a course: added `subject_id`, backfilled where one course's claims cite the source (49 of 49 on the dev copy).
- `review_events` is append-only and points at concepts, cards and claims, so courses are archived and claims soft-deleted, never removed.
- `internal/cite.Verify` needs a quote of 20+ characters (or a whole display formula). Fakes use it for anchors, so there is still one verifier.
- `api/openapi_test.go` validates the spec with kin-openapi and checks domain JSON against it. Flow-style YAML descriptions with commas must be quoted.
- `internal/app/handlers_test.go` fails when a spec path has no handler test unless it is on `notServed`. That file belongs to P1-API.
- `teach.Store` and `domain.TeachStore` are pinned to the same method set, so `LoadConcept` stays unscoped; P1-API checks ownership before calling it.

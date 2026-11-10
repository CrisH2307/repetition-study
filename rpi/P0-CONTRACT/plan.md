# P0-CONTRACT plan

1. domain/types.go: `ReportStatus` (`checked`, `unchecked`), `Report.Status`, `ReviewEvent` gains
   `ClaimID`, `SessionID`, `SelfGrade`, `CheckerModel`, `CheckerVersion` (all nullable), `DueItem.Reason`,
   `BlockKind` + `SourceBlock`, `Pack`.
2. domain/interfaces.go: `TeachStore` (same method set as `teach.Store`), `ReportReader.GetReport`,
   `SessionScheduler.ApplyTeachBackSession`, `PackLoader.LoadPack`. Existing interfaces unchanged.
3. domain/contract_test.go: `teach.Store` and `domain.TeachStore` assign both ways.
4. migrations/0003_contract_v2.sql + test: unique teach_back per session, new columns, trigger still rejects.
5. fakes: TeachEngine.GetReport, Scheduler.ApplyTeachBackSession (idempotent), TeachStore, PackLoader; tests.
6. api/openapi.yaml new fields; api/openapi_test.go validates the doc and checks domain JSON against schemas.
7. Run build, vet, race tests; clean DB `make migrate test`; upgrade path on a DB with data.

Risks: `teach.Engine.Finish` returns `Status: ""` until P0-OUTAGE sets it; spec says required enum.

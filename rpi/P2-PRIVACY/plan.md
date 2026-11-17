# P2-PRIVACY plan
1. internal/store/privacy.go: ExportUser (one read-only snapshot), DeleteUser (one tx, set_config local, 23 ordered deletes, then the user row).
2. internal/store/privacy_test.go: one populated user per table, three users, counts before and after.
Risks: a table added later is missed by delete (FK error rolls back loudly, test table list must grow).

# P2-API plan
1. v4.go: seven handlers behind sign-in; Deps gets Feedback, Privacy, Imports, Importer; Wire fills them.
2. owner.go: spendRec copies meter spend into the ledger after each call; capLLM reads MonthTotal before each check and fails closed.
3. metrics.AddCost + CSV block: model spend per checked session by month.
4. Tests: fake stack spec test, v4 isolation, delete end to end, cap from Postgres ledger, restart, export cost.
Risks: a week that spans two months counts in the Monday's month (ponytail note in metrics.go).

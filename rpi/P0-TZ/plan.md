# P0-TZ plan
1. Config.Loc + LocalWeekStart; place() builds each day with time.Date in Loc (DST safe).
2. Store.userConfig reads users.time_zone; Generate and Progress use it. Unknown zone = error.
3. Tests: pure DST test (2026-03-08, 2026-11-01), store test with Toronto and a bad zone.
Risk: a week crossing DST is 167 or 169 hours; ranges use AddDate in Loc so they stay correct.

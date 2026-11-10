# P0-TZ research
- users.time_zone (migration 0003, text, default America/Toronto, no SQL check). Validated in Go with time.LoadLocation.
- Day math was UTC in allocate.go place() (weekStart + d*24h + 9h) and store.go/progress.go (.UTC()).
- Unverified: callers pass weekStart as a UTC midnight Monday. We read only its Y/M/D and rebuild local midnight.

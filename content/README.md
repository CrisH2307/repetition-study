# content

Course material is never committed (rule D7). Real files live in `content/private/`, which is gitignored.

Put these there (they are generated or copied by hand):
- `private/source.md`: the parsed course PDF (`scripts/parse_pdf.py`).
- `private/unit.json`: the full unit, with `sources[0].text` filled in.
- `private/seed_pack.sql`: built by `python3 scripts/parse_pdf.py --sql content/private/unit.json content/private/seed_pack.sql`.
- `private/damaged-formulas.md`: notes on formulas to fix by hand.

`make seed` loads `scripts/seed.sql` only. For the real course, load `content/private/seed_pack.sql` and then `content/seed.sql` yourself.

Tracked here: `fixture/` (our own short notes and unit, used by CI), `unit.json` (concepts and claims, source text left empty), `seed.sql`, `allowlist.json`.
Tests: `go test ./content/` always checks the fixture; private checks skip when `content/private/` is missing.

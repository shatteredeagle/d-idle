# First hour: the daily table tool

Why: the headline metric of the project review (docs/research/project-review-2026-11-10.md, R2, sections 3 and 5): a fresh hero, 60 realtime minutes at 1x, one row a day in [probes/firsthour.md](probes/firsthour.md). TASK-245.

## Use
- Fixture dry run (V: unit tests, `go test ./cmd/d2firsthour`): `tools/firsthour.sh --class paladin --log cmd/d2firsthour/testdata/hour.txt --dry-run`.
- Real run, after the sandbox hero has played 60 minutes from a reset (U: not run live; no game was started): `tools/firsthour.sh --class paladin` fetches `docker logs $D2_SANDBOX_CONTAINER` through the worker host, computes the row, appends it. `--start RFC3339` names the hero's first moment when the container log holds older heroes; the default is the first `d2bot start` line.
- Exit 1 means the log does not hold a full fresh hour (shorter than 60 min minus 90 s, first minute line above clvl 2, no minute lines, no version stamp); nothing is written. It never guesses.

## Fields and how each is derived
The tag table is the header of probes/firsthour.md (V read from a log line, S a stated rule, U the log may not carry it). Lines used: `d2bot start` (version, V: cmd/d2bot/main.go), `minute` (clvl, kills, deaths, gold, restarts, stashGold; V: docs/lifelog.md and internal/d2dsoakstat/testdata), `zone entry`, `safety decision` (docs/lifelog.md), `poverty mode: on|off` (internal/bot/leveling.go). Build swaps show as `a > b` in the build column, so a day with a redeploy mid-hour is visible.

## Dead ends and open points
- U: chickens decided inside the fight loop are not logged (docs/lifelog.md), so the chicken count is a floor.
- U: the shuttle rule (A,B,A over three consecutive non-town entries) and its cap are a proposal; town ids 1, 40, 75, 103, 109 are S.
- Session record and ledger files of the data volume are not read: the log carries every number above; no credential-bearing path is touched.
- docs/dev-reset.md (agent/devreset) was not merged when this was written; the reset gives the fresh hero whose first `d2bot start` line starts the hour. No file overlap.

## Live format check (2026-11-10, read-only `docker logs` of the sandbox container, V)
All five line kinds match docs/lifelog.md as JSON slog (`d2bot start` with `version`, `minute` with `clvl kills gold deaths restarts stashGold`, `zone entry` `area`, `safety decision` `action=chicken`, `poverty mode: on|off`); the parser needed no change. The log holds 18 `d2bot start` lines over the container life and the hero was clvl 10 mid-session, so the dry run failed loudly as designed ("not a fresh hero"). Use `--start` for the fresh hero's first moment after a reset; the default (first start line) only fits a log that begins at the reset. `kills` in `minute` is a running total (V: 2850 at clvl 10), so the reset-summing rule only matters across restarts.

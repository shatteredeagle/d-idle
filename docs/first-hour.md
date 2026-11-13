# First hour: the daily table tool

Why: the headline metric of the project review (docs/research/project-review-2026-11-10.md, R2, sections 3 and 5): a fresh hero, 60 realtime minutes at 1x, one row a day in [probes/firsthour.md](probes/firsthour.md). TASK-245.

## Targets (user decision 2026-11-13, "recalibrate the headline target")
The old implicit target, the d2-sme's "a competent human is clvl 13-15 at 60 minutes", is WRONG and retired: clvl 13 needs 112,725 XP (clvl 15: 176,132), and no Act 1 Normal kill pays more than 76 XP (V run, docs/research/xp-gold-trace.md sections 5 and 6; the award function matches game.exe on 119,790 of 119,790 grid lines). The SME's other advice (areas, potion buffer, melee entry) stands. Tiers for a fresh Paladin, 60 realtime minutes at 1x; XP thresholds are D (Experience.txt, xp-gold-trace Q6): clvl 5 = 7,875, 6 = 14,175, 7 = 22,680, 8 = 32,886, 9 = 44,396.

| measure | floor | target | stretch | tag |
|---|---|---|---|---|
| clvl@60 | 5 | 7 | 8 to 9 | V tables and model: observed runs clvl 6/5/5; ideal route (Cold Plains from minute 15, 14.6 kills/min) = 26,020 XP = clvl 7; 25 kills/min with Stony Field = clvl 9 to 10, so 8 to 9 is a haircut (U how much) |
| kills per minute (kills@60) | 6 (350) | at least 14 in the field (about 750) | 20 or more (about 1,000) | V rates from hour1-analysis section 5; totals are S arithmetic from the xp-gold-trace section 5 schedules |
| route | Den done, Cold Plains reached | Cold Plains entered by minute 15 | Stony Field entered by minute 40 | S from the schedules (xp-gold-trace section 5); the minute marks are U (a judgement) |
| chickens | 0 | 0 | 0 | S: the user's floor; run 3 had 5 (potion starvation, belt fix merged); inside-fight chickens are not logged (U), so the count is a floor |
| deaths | 0 | 0 | 0 | V (three runs: 0) |
| zone entries / shuttles | at most 60 / 10 | at most 35 / 5 | at most 25 / 3 | U: runs had 33/7, 78/49, 56/5; the caps are judgement |
| town share of the hour | at most 12 % | 5 to 8 % | 5 % | U: judgement; the town trips of run 1 are in hour1-analysis section 2 |
| gold | not a target | not a target | not a target | V: gold is not a lever in the early game (0.39 gold per Blood Moor kill, 0.61 Cold Plains, 1.4 Stony Field; 528 kills are about 300 gold, xp-gold-trace section 3); the column is reported, never tuned to |

Reading: clvl is the one number the tables settle (V); the rest are rates and habits the tables only bound. Past clvl 7 the starter areas are exhausted (Blood Moor 17.1 XP per kill at clvl 7, 13.4 at 8), so each further level needs Stony Field or later. Rows of the three runs against these tiers: [probes/firsthour.md](probes/firsthour.md) (Targets section).

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

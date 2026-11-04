# farmfallback: the gold farm as data, with a fallback

Decided 2026-11-04 (no-area, recommendations).

## What changed
- `farm: true` per step in `progression-normal.json` (mirrored in `docs/plan/data/`) replaces `merc.FarmAreas`
  (removed). Flagged: Blood Moor 2, Cold Plains 3, Stony Field 4, Den 8, Cave 1 9, Cave 2 13, Burial Grounds 17,
  Crypt 18 (today's set, read from the code) plus Dark Wood 5 and Black Marsh 6 (merc-less clvl 13-16: 0 deaths in
  315 min over 3 seeds, S). Missing flag = not a farm area. `Plan.FarmAreas(diff)` errors on a missing table; the
  farm is then skipped with a warning, never a built-in set.
- `goal.go` picker: the farm is a preference. When the restricted pick is empty (every farm area banned, cleared or
  excluded) it logs `farm: restricted pick empty, falling back` and picks again among all areas with the normal
  exclusions. `guardedAreas` (leveling.go, broke.go) keeps the restriction without a fallback.
- Not changed: time budget (mercrevive2), cap 17, revive always on.

## Tests
`internal/bot/farmfallback_test.go`: flag from data (embedded and a second table), missing table error, farm area
available (unchanged), DW/BM eligible at clvl 13-16, all banned -> fallback. The picker closure itself is mirrored
in the test, not driven end to end (U). The plan is per difficulty, not per build, so "two builds" is two tables.

## Live rerun check (not run)
Gate 2 seed with a dead merc at clvl 15: no `no eligible` flood, DW/BM picks, at most one fallback line per game.
Measure first (SME list): deaths per hour in DW/BM merc-less, gold per hour in the fallback, count of fallback lines.

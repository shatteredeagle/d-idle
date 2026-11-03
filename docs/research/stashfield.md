# stashGold on the minute line

Why: BELTFIX5 (branch agent/beltfix5) found the clvl 11 belt window of Gate 1 run gate1-20261103-0338 was a bot defect: 1418 gold sat in the stash, 6 in the pack. The minute line had only `gold` (pack), so judge and analysts saw poverty.

## What changed
- Bot minute line (internal/bot/observe.go): additive `stashGold`, the same player stat (`d2.Player.StashGold`, stat 15) the autopsy and the merc log use for stash gold. V: compiles and vets; not run live.
- Schema is additive: old archives lack the key; `d2dsoakstat.Minute.HasStash` is false and `StashGold` 0 there.
- `d2dsoak` prints, under each seed, one line per belt window over 5 virtual minutes (`BeltReportMin`): min/max pack gold, min/max pack+stash gold, and a tag: `poverty (pack+stash < potion price)` when the best minute stays under `PotionPrice` (30, Minor Healing), `funded (...)` otherwise, `stash unknown (old archive)` when no minute carries the field. Report only: the BELT invariant, its limit and the verdict are untouched.
- Tests: internal/d2dsoakstat/invariants_test.go (TestBeltWindowGoldReport: poverty, funded, old archive; TestMinuteStashGold).

## Open
- Whether BELT should treat a funded window as a bot fault rather than poverty is a spec delta (soak-gates) and the user's decision; not made here.
- The first minute line of a log is never counted in a window (existing loop starts at the second line), so a window of N lines reports N-1 when it starts on line one.

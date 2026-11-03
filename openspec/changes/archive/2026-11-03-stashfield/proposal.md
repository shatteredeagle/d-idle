## Why

BELTFIX5 found the clvl 11 belt window of gate1-20261103-0338 was a bot defect (1418 gold in the stash, 6 in the pack). The minute line only showed pack gold, so the judge could not tell poverty from a bot that did not withdraw.

## What Changes

- Minute line: additive `stashGold` field. Old archives lack it and read as 0 (stash unknown).
- d2dsoak: report-only belt window gold line (pack and pack+stash range, poverty or funded tag against PotionPrice 30). The BELT invariant is unchanged; the spec delta is a user decision.

## Impact

internal/bot/observe.go, internal/d2dsoakstat, docs. No spec delta.

## Overlap (declared)

The same files are listed for specmin (soak-minutes-virtual), a finished card; this card only adds the stashGold field and a report line to them.

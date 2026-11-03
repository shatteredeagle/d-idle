# tomestat: the tome's stat 70 lost in the field (Gate 2)

## Why
TOWNSHOP found that the field read drops the Tome of Town Portal's stat 70 while the town read gives 13 to 19 (docs/townshop.md). Cause (docs/tomestat.md): `d2dsrc.Extras.Items` was keyed by unit id and held carried AND ground items. d2d numbers ground items with the loot table's own counter and carried items with the character's counter, both from 1, so a ground item with the tome's id overwrote the tome's extras (no quantity). The field has ground items, the town hardly any.

## What Changes
- `Extras.Ground` (new): ground items by their id. `Extras.Items` holds carried items only. d2dloot reads `Extras.Ground`.
- Regression test with a ground item sharing the id of a carried stack.

## Overlap (declared)
Overlap (declared): `internal/d2dsrc/source.go`, `internal/d2dsrc/items.go`, `internal/d2dloot/d2dloot.go` and its test belong to earlier cards; this card changes a map and a field there and does not list them in owns. townshop (docs/townshop.md, shop budget) and fixprogress (town chores) are the consumers of the verdict; no file shared.

## Not done
Live check (U). The debounce clock (virtual seconds, wrong at high speed) and id -1 (unassigned) collisions are left (docs/tomestat.md).

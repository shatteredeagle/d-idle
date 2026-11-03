## Why

Gate 1 ms1 seed 17 failed on a bot bug: at clvl 9 in the Blood Moor, empty tome, 0 healing potions, 284 gold, the emergency town trip failed 193 times with `nav: no exit to that level: level 1 from 2` and only the failsafe ladder got the character home (docs/soaks/gate1-multiseed-20261103-1226-ms1.md, seed 17 analysis).

## What Changes

- d2dworld keeps a border once seen (`keepBorders`): a later probe round that finds nothing for a neighbour no longer erases it, so the Blood Moor keeps its town gate.
- town: an empty tome now counts as no portal source; the first scroll is bought before the kit potions when the gold still covers scroll plus two potions.
- docs/navexit21.md: findings, tags, live checklist.

## Overlap (declared)

fixprogress also owns internal/town/chores.go. This card changes only the two lines of shopVisit that call buyFirstTP (one new first call before the kit potions); the owners touch different functions, so the merges are independent.

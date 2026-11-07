# PR draft: a character enters a game with the life and mana it was saved with

> status: DRAFT, NOT POSTED. Nothing is pushed upstream, posted or filed (no user go). Branch `pilot/load-life-mana` (commit 693f9def, on the private fork only). Proof so far: [fid-loadhp-trace.md](fid-loadhp-trace.md). Not ready for upstream: the differential against game.exe's code is missing (section 4 of the trace).

````
## What changes
`World::take_character` no longer refills life and mana to the maximum when a character enters a game. It keeps the saved current value with its fractional bits dropped, never above the maximum. Stamina is unchanged (still refilled): game.exe's rule for it is not shown.

Why: commit 756bf206 refilled all three ("D2's save and exit to heal") because a save left mid-fight started the next game near death. On game.exe 1.14d a save keeps its life and mana: six loads of two characters (hurt by 2 and 12 percent of life, 50 and 0 percent of mana): file 89.54 / 77.76 life, game.exe 89 / 77, d2d 91 / 88 (full).

## Evidence
| Character | stat | file | game.exe (3 of 3) | d2d before (3 of 3) | d2d after (rule applied by hand, not run on the save) |
|---|---|---|---|---|---|
| Sorceress clvl 6 | life | 89.54 of 91 | 89 | 91 | 89 |
| Sorceress clvl 6 | mana | 24.39 of 45 | 25 (+regen) | 45 | 24 |
| Paladin clvl 4 | life | 77.76 of 88 | 77 | 88 | 77 |
| Paladin clvl 4 | mana | 19.5 of 19.5 | 19 | 19 | 19 |

Test: `tests/test_game.cpp` block "A saved character keeps its life and mana": fails before the change, passes after (worker host, `test_game`).

## Not claimed
The Paladin maximum mana (19.0 in d2d against 19.5) is a separate cause. Stamina, a save at zero life and a dead character are not shown.
````

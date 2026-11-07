# PR draft: a character enters a game with the life and mana it was saved with

> status: DRAFT, NOT POSTED. Candidate branch `local/fid-load-life-mana-r1` (head 76013e22, one commit on upstream main da2d96cf, private fork remote only). The older pilot branch `pilot/load-life-mana` (693f9def) is superseded. NOT READY to open: no game.exe address for the load rule, no oracle script, no independent verification (docs/research/pr-bundle-queue.md, "Upstream rules as of 2026-11-07"). Proof so far: [fid-loadhp-trace.md](fid-loadhp-trace.md). The body below follows the upstream `.github/pull_request_template.md`.

````
## What changes
d2d refilled a character's life and mana to the maximum when it entered a game. game.exe keeps the value the save held, with the fraction dropped. `World::take_character` now does the same, never above the maximum. Stamina is unchanged (still refilled): game.exe's rule for it is not shown.

## Evidence from game.exe 1.14d
| Address | What it shows |
|---|---|
| (none yet) | The rule is shown only by live reads of game.exe. To fill in before opening: the function that writes stats 6 and 8 from the d2s on entering a game (the join path is FUN_0052d440 -> FUN_0055df00 -> FUN_0055dbc0, docs/research/re/net-join.md) |

Live reads (game.exe, 3 of 3 each; 12 loads of two characters in all, read 1 to 2 s after the load):

| Character | stat | file | game.exe | d2d before | d2d after |
|---|---|---|---|---|---|
| Sorceress clvl 6 | life | 89.54 of 91 | 89 | 91 | 89 |
| Sorceress clvl 6 | mana | 24.39 of 45 | 25 (regen included) | 45 | 24 |
| Paladin clvl 4 | life | 77.76 of 88 | 77 | 88 | 77 (by the rule, not run on the save) |
| Paladin clvl 4 | mana | 19.5 of 19.5 | 19 | 19 | 19 |

## Claims (a batch only)
| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| 76013e22 | A save's life and mana are kept at load, fraction dropped, capped at the maximum | live reads above; address missing | `test_game` prints "OK: a save's life and mana are kept ..." |

## How to re-check
```sh
python3 tools/pr_guard.py
cmake --build build --target test_game && ./build/tests/test_game   # D2_MPQ_DIR set
```
The test fails on the parent commit (`Assertion hero.character.stats.values[kLife] == 89 * 256 failed`) and passes with the change.

## Checklist
- [x] A test in `tests/`
- [ ] The finding in `docs/research/re/` (to add with the address)
- [x] `python3 tools/pr_guard.py` passes (0 FAIL, 0 WARN); `presubmit.py --quick` 0 FAIL, 2 WARN (docs-sync: answered by this body; build skipped by --quick)

Honest limits: no differential against game.exe's own load function (not yet identified), only 12 live loads on two characters hurt by at most 12 percent of life and 50 percent of mana; stamina, a save at zero life and a dead character are not shown; the Paladin maximum mana (19.0 in d2d against 19.5) is a separate cause (TASK-102), not claimed; the fraction drop is seen 1 to 2 s after load, so a first-frame normalisation is not excluded.

---

Agent: Claude Code
````

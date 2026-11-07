# PR draft: a character enters a game with the life and mana it was saved with

> status: DRAFT, NOT POSTED. Candidate branch `local/fid-load-life-mana-r2` (one commit on upstream main da2d96cf; scratch clone and a bundle on the worker host, the push to the private remote is still to do). SUPERSEDES r1 (76013e22, "fraction dropped", refuted: the fraction was the client's display copy). NEEDS independent Opus verification before opening (docs/research/pr-bundle-queue.md). Proof and limits: [fid-loadhp-trace.md](fid-loadhp-trace.md). The body below follows the upstream `.github/pull_request_template.md`.

````
## What changes
d2d refilled a character's life and mana to the maximum when it entered a game. game.exe sets them back from the save, exactly (the fraction included, no clamp to the maximum). `World::take_character` now leaves them alone. Stamina is unchanged: game.exe sets it to the maximum, as d2d already did.

## Evidence from game.exe 1.14d
| Address | What it shows |
|---|---|
| FUN_00534330 0x534364, 0x534387 | a save of version above 0x5b is loaded by FUN_0056b180 |
| FUN_0056b180 0x56b270 | the stat block is decoded first (FUN_0056a620 -> FUN_0056a4f0: 9-bit id, save bits of ItemStatCost, FUN_00627260 per stat, raw 256ths) |
| FUN_0056b180 0x56b283, 0x56b28f | stats 6 (life) and 8 (mana) are read into two locals right after the decode |
| FUN_0056b180 0x56b3c3 -> FUN_0056af80 | the restore, after skills, items and the mercenary |
| FUN_0056af80 0x56b0ff, 0x56b10d | stat 6 and stat 8 are set to those two locals with FUN_00627260; there is no comparison with the maximum |
| FUN_0056af80 0x56b01d | stamina (stat 0xa) is set to the maximum (FUN_00625db0 = stat 0xb) |
| FUN_00548760 0x5487b7; FUN_0045d9b0 0x45d9d8 | the server sends life, mana and stamina as whole points (`sar 8`); the client's player unit sets `value << 8`. This is why a read of the client unit shows 89 for a saved 89.54 |

Oracle `tools/emu/fl_load.py` runs the real FUN_0056b180 (stat decode, setter, getter, FUN_0056af80) on 414 stat blocks and `test_game --load-stats` runs `World::enter` on the same values; `diff` of the two: d2d before this change 392 of 414 differ, with it 0. Live reads of the client unit (game.exe, 3 loads each, 12 in all, read 1 to 2 s after the load) are the whole-point floors of the oracle's values:

| Character | stat | file | game.exe server (oracle) | game.exe client (oracle, and live) | d2d before | d2d after |
|---|---|---|---|---|---|---|
| Sorceress clvl 6 | life | 89.54 of 91 | 22921 / 256 | 89 (live 89) | 91 | 22921 / 256 |
| Sorceress clvl 6 | mana | 24.39 of 45 | 6244 / 256 | 24 (live 25, regeneration) | 45 | 6244 / 256 |
| Paladin clvl 4 | life | 77.76 of 88 | 19907 / 256 | 77 (live 77) | 88 | 19907 / 256 |
| Paladin clvl 4 | mana | 19.5 of 19.5 | 4992 / 256 | 19 (live 19) | 19.5 | 4992 / 256 |

## Claims (a batch only)
| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| this one | A save's life and mana are kept exactly at load, no clamp; stamina is the maximum | the addresses above; oracle 414 cases, 0 differ | `test_game --load-stats` against `uv run python tools/emu/fl_load.py game`; `test_game` prints "OK: a save's life and mana are kept ..." |

## How to re-check
```sh
python3 tools/pr_guard.py
cd tools/emu && uv sync
uv run python fl_load.py gen 400 > /tmp/cases.tsv
uv run python fl_load.py game /tmp/cases.tsv > /tmp/game.tsv
../../build/tests/test_game --load-stats /tmp/cases.tsv | grep -E '^[A-Za-z_]\w* -?[0-9]+( -?[0-9]+){5}$' > /tmp/d2d.tsv
diff /tmp/game.tsv /tmp/d2d.tsv && echo same
```
`test_game` fails on the parent commit (`Assertion hero.character.stats.values[kLife] == 89 * 256 + 138 failed`) and passes with the change.

## Checklist
- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same commit
- [ ] The finding in `docs/research/re/` (to add: the loader row for FUN_0056b180 / FUN_0056af80)
- [x] `python3 tools/pr_guard.py` passes (0 FAIL, 0 WARN); `presubmit.py --quick` 0 FAIL, 3 WARN (docs-sync: answered by this body; build and gcc skipped by --quick)

Honest limits: the oracle runs the stat decode and the restore with the rest of the file hooked (header, skills, items, corpse, mercenary), on a unit with no gear; the server's first frames after the load are not run, nor the stat-change callback FUN_0055b800 that scales current life when a maximum changes; no live read of the server's unit exists; a save at zero life is kept as zero but what the game does with a dead character on entry is not traced; saves of version 0x5b and below (FUN_00534020) and the decode branch for 0x5c..0x5e are not covered; the Paladin maximum mana (19.0 in an earlier probe) is not from the load (the maximum is kept exactly) and is not claimed.

---

Agent: Claude Code
````

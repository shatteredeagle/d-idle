# Draft (local, NOT posted): "Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)"

> status: ready for the independent verifier (prprep, 2026-11-11). Branch `fid/cold-enchanted-length-r2` on the private fork remote, head `05d5d754`, two commits on upstream main `b6b59ae9` (replaces `fid/cold-enchanted-length`, head b3c6341a on a5b970d5; same change, rebased, plus a new glue test; the earlier push `fid/cold-enchanted-length-r1` (98e10ccf) is stale: a pr_guard warning in the oracle docstring was fixed and a force-push is not allowed here). Trace: [../../probes/fid/fid-boss-mods.md](../../probes/fid/fid-boss-mods.md) and [../../research/monster-mods-trace.md](../../research/monster-mods-trace.md).

Nothing here is posted. PR title: `Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)`

## What changes

A Cold Enchanted champion, unique or super unique carries a cold element on its blow. d2d put that cold on the blow with a length of 0 (`stats.elem_len` was only set for Ghostly), so the chill lasted one frame. game.exe's init FUN_005a1c70 adds the unit's level x 5 + 100 frames to the cold length (stat 0x38). `rules::BossStats` gets `elem_len_per_level` (5 for Cold Enchanted, base 100), `rules::boss_elem_length(stats, level)` computes it (a level above 110 counts as 110, the cap the oracle shows), and `make_boss` uses it instead of the bare `elem_len`. Ghostly's 150 is unchanged.

Two commits: the oracle `tools/emu/umod_stats.py` with its d2d side `test_umod_stats` (it shows the difference on the parent), then the claim (+8 / -2 lines in `ai.cpp` and `uniques.hpp`, a doc paragraph in `docs/research/re/monsters.md`, and `tests/game/test_boss_cold_length.cpp`, which runs the real `make_boss`).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005a2120` | the spawn init that runs the fixed mods and the unit's own mods through the table at `0x73c008` |
| `FUN_005a1c70` | Cold Enchanted's init: cold min / max by MonLvl damage x the mod's percent, and the cold length = the unit's level x 5 + 100 |
| `FUN_005a1080` | Ghostly's init: cold length 150 (unchanged here) |
| `FUN_005a00f0`, `FUN_005a0000`, `FUN_00611d30` | the MonUMod constants, the MonLvl row and the difficulty row the inits read (run for real by the oracle) |

## Counted differentials (V run, the worker host)

`tools/emu/umod_stats.py` runs game.exe's real `FUN_005a2120` and the mods' inits on a synthetic unit (every table loaded by game.exe itself); `test_umod_stats --umod-stats` runs `rules::boss_stats` on the same cases the way `make_boss` applies it. `umod_stats.py --compare` lists the fields that differ. Seed 1, 157,618 cases (all single mods first, then random combinations, kinds champion / unique / super unique / minion).

| cold_len mismatches, grouped | parent (oracle commit only) | with the claim |
|---|---|---|
| cases whose elemental mod is Cold only (no Fire / Lightning Enchanted) | 7,869 | 0 |
| Cold together with Fire or Lightning Enchanted | 3,124 | 1,625 |
| minions (d2d tags a minion without mods) | 5,657 | 5,657 |
| unlisted combinations (5 or more mods) | 2,290 | 248 |

The remaining rows are not this claim: `BossStats` keeps ONE element (`stats.elem`, the last Enchanted mod in the list wins), so a unique with two elemental mods loses an element in d2d whatever the length; a minion has no mods in d2d (README row of `umod_stats.py`). Both are earlier, separate deviations. The first row is the claim: every case whose only elemental mod is Cold now matches.

Glue test: `test_boss_cold_length` (real `make_boss`, 3 rows x 3 difficulties x Cold and Fire Enchanted) asserts the cold element's length is `100 + 5 * min(level, 110)` and the fire element's 0. It fails on the parent (assertion on the `dur` line) and passes with the claim; run with the 1.14d data and patch installer.

Neighbours with the claim: `test_monsters` rc 0, `test_game` rc 0. `boss_level.py` against `test_boss_level` (the merged level / life oracle): 136,624 cases, 5,973 mismatching, all in `life256` (e.g. quillrat1, champion, mod 39); that field is not touched by this change (U: it is a separate difference on current main, the same count is expected on the parent; not run on the parent in this prep).

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build --target test_umod_stats test_boss_cold_length test_monsters test_game
cd tools/emu && uv sync
uv run python umod_stats.py 100000 1 > /tmp/game.tsv
../../build/tests/test_umod_stats --umod-stats /tmp/game.tsv > /tmp/d2d.tsv
uv run python umod_stats.py --compare /tmp/game.tsv /tmp/d2d.tsv | grep cold_len
../../build/tests/test_boss_cold_length
```

## Honest limits (to put in the body)

- The level the length uses is the unit's level after the fixed mods (`monster.stats.level` in `make_boss`). The oracle's synthetic unit starts at the MonStats level for the difficulty and adds the level mod; the real-`make_boss` test checks the same formula on the unit's own level, not against game.exe's number for an area-placed unit.
- The clamp at 110: the oracle shows 650 frames for the level-113 units; the clamp is the observed cap of the 1.14d table path, its source in the disassembly was not read (U).
- Unique mods with two elements (see above) and minion mods stay as they are.
- Overlap check (read-only `gh`, 2026-11-11): open #371 (unique mods 21, 35, 40-42, `components/rules/uniques.hpp`) and #401 (`ai.cpp`) touch neighbouring lines; a rebase on whichever merges first is needed, no semantic overlap.

## Checklist

- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same pull request (oracle commit first, claim second)
- [x] The finding in `docs/research/re/monsters.md`
- [x] `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN (the one earlier WARN, FUN_005a1910 in the oracle's docstring, was fixed: LAB_005a1910)
- [x] `python3 tools/presubmit.py --quick --base upstream/main`: see the result line at the end of this file (run by `pp_cold_push.sh`)

Agent: Claude Code

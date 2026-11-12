## What changes

A Cold Enchanted champion, unique or super unique carries a cold element on its blow. d2d put that cold on the blow with a length of 0 (`stats.elem_len` was only set for Ghostly), so the chill lasted one frame. game.exe's init FUN_005a1c70 adds the unit's level x 5 + 100 frames to the cold length (stat 0x38). `rules::BossStats` gets `elem_len_per_level` (5 for Cold Enchanted, base 100), `rules::boss_elem_length(stats, level)` computes it, and `make_boss` uses it instead of the bare `elem_len`. Ghostly's 150 is unchanged.

The level is taken as at least 1 and at most MonLvl's last row (`0x5a1cf5..0x5a1d1c`, rows at table +0xb74: 111 rows, so 110); `std::clamp(level, 1, max_level)` with `max_level` = the loaded MonLvl table's rows - 1 (`boss_elem_length` takes it as a parameter; `make_boss` passes `game_data.monsters.lvl.size() - 1`). The length is ADDED to whatever stat 0x38 the unit had (`lea eax, [eax+eax*4+0x64]` at `0x5a1dcd`, then the adder FUN_006272b0), so a unit with both Ghostly and Cold would carry 150 plus the Cold part; d2d keeps one `elem_len`, the last mod (whether a real boss carries both is U).

Two commits: the oracle `tools/emu/umod_stats.py` with its d2d side `test_umod_stats` (builds on its own, prints `boss.elem_len` as `make_boss` used it before; it shows the difference on the parent), then the claim (`ai.cpp`, `uniques.hpp`, `test_umod_stats` switched to `boss_elem_length`, a doc paragraph in `docs/research/re/monsters.md`, and `tests/game/test_boss_cold_length.cpp`, which runs the real `make_boss`).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005a2120` | the spawn init that runs the fixed mods and the unit's own mods through the table at `0x73c008` |
| `FUN_005a1c70` `0x5a1ce5..0x5a1d1c` | the unit's level (stat 0xc), at least 1, at most MonLvl's rows - 1 |
| `FUN_005a1c70` `0x5a1dcd` | cold length: stat 0x38 += level x 5 + 100 |
| `FUN_005a1080` | Ghostly's init: cold length 150 (unchanged here) |
| `FUN_005a00f0`, `FUN_005a0000`, `FUN_00611d30` | the MonUMod constants, the MonLvl row and the difficulty row the inits read (run for real by the oracle) |

## Counted differentials (V run, the worker host, tree = upstream `62732598` plus the branch)

`umod_stats.py` runs game.exe's real `FUN_005a2120` and the mods' inits on a synthetic unit; `test_umod_stats --umod-stats` runs `rules::boss_stats` on the same cases the way `make_boss` applies it; `--compare` lists the fields that differ. `cold_len` mismatches, grouped by the mods of each case (a case is Cold Enchanted with no Fire / Lightning Enchanted, Cold with one of them, a minion, or five or more mods):

| Group | seed 1 (157,618 cases): parent | claim | seed 9 (107,618 cases): parent | claim |
|---|---|---|---|---|
| Cold Enchanted, no Fire / Lightning Enchanted | 7,869 | 0 | 4,745 | 0 |
| Cold together with Fire or Lightning Enchanted | 3,124 | 1,625 | 1,591 | 828 |
| minions of a Cold Enchanted boss | 5,657 | 5,657 | 3,732 | 3,732 |
| unlisted combinations (5 or more mods) | 2,290 | 248 | 1,154 | 135 |

The remaining rows are two earlier, separate deviations: `BossStats` keeps ONE element (`stats.elem`, the last Enchanted mod wins), and a minion has no mods in d2d (README row of `umod_stats.py`).

Glue test: `test_boss_cold_length` (real `make_boss`, 3 rows x 3 difficulties x Cold and Fire Enchanted) asserts the cold element's length is `100 + 5 * clamp(level, 1, MonLvl rows - 1)` and the fire element's 0. It fails on the parent and passes with the claim. `test_monsters` and `test_game`: rc 0.

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

- The real-`make_boss` test checks the formula on the unit's own level; the oracle's synthetic unit starts at the MonStats level for the difficulty. Both give the same rule; neither is an area-placed unit compared against game.exe's number.
- Two-element uniques and minion mods stay as they are (separate deviations, rows above).
- Overlap check (read-only `gh`, 2026-11-11): open #371 (unique mods 21, 35, 40-42, `uniques.hpp`) and #401 (`ai.cpp`) touch neighbouring lines without touching `elem_len` or FUN_005a1c70.
- `test_umod_stats` is changed in the claim commit on purpose: it switches from the bare `elem_len` to `boss_elem_length`, the rule it checks.

## Checklist

- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same pull request (oracle commit first and building alone, claim second)
- [x] The finding in `docs/research/re/monsters.md`
- [x] `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN
- [x] `python3 tools/presubmit.py --quick --base upstream/main`: 0 FAIL, 1 WARN (build skipped by `--quick`; built and tested by hand)
- [x] `ctest -j8` on the full build: the same 4 failures as upstream main on my box (`test_mpq`, `smoke_d2d`, `quests_act1`, `quests_act4`: no video MPQs here), nothing else

---

Agent: Claude Code

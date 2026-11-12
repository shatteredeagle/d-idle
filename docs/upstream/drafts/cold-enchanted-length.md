# Draft (local, NOT posted): "Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)"

> status: r4, ready for the verifier's second pass (prprep2, 2026-11-11). Branch `fid/cold-enchanted-length-r4` on the private fork remote, head `30713b60`, two commits on upstream main `62732598`. Supersedes `-r1` (98e10ccf), `-r2` (05d5d754) and `-r3` (1b4036a3), stale on the remote (no force-push). Verifier report: [../verification-2026-11-11.md](../verification-2026-11-11.md) (r3: PASS on the evidence, one fix: the oracle commit did not build on its own; applied here). Trace: [../../probes/fid/fid-boss-mods.md](../../probes/fid/fid-boss-mods.md) and [../../research/monster-mods-trace.md](../../research/monster-mods-trace.md).

Nothing here is posted. PR title: `Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)`

## What changes

A Cold Enchanted champion, unique or super unique carries a cold element on its blow. d2d put that cold on the blow with a length of 0 (`stats.elem_len` was only set for Ghostly), so the chill lasted one frame. game.exe's init FUN_005a1c70 adds the unit's level x 5 + 100 frames to the cold length (stat 0x38). `rules::BossStats` gets `elem_len_per_level` (5 for Cold Enchanted, base 100), `rules::boss_elem_length(stats, level)` computes it, and `make_boss` uses it instead of the bare `elem_len`. Ghostly's 150 is unchanged.

The level is taken as at least 1 and at most MonLvl's last row (`0x5a1cf5..0x5a1d1c`, rows at table +0xb74: 111 rows, so 110); `std::clamp(level, 1, 110)`. The length is ADDED to whatever stat 0x38 the unit had (verifier V read: `lea eax, [eax+eax*4+0x64]` at `0x5a1dcd`, then the adder FUN_006272b0), so a unit with both Ghostly and Cold would carry 150 plus the Cold part; d2d keeps one `elem_len`, the last mod (whether a real boss carries both is U).

Two commits: the oracle `tools/emu/umod_stats.py` with its d2d side `test_umod_stats` (builds on its own, prints `boss.elem_len` as `make_boss` used it before; it shows the difference on the parent), then the claim (`ai.cpp`, `uniques.hpp`, `test_umod_stats` switched to `boss_elem_length`, a doc paragraph in `docs/research/re/monsters.md`, and `tests/game/test_boss_cold_length.cpp`, which runs the real `make_boss`).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005a2120` | the spawn init that runs the fixed mods and the unit's own mods through the table at `0x73c008` |
| `FUN_005a1c70` `0x5a1ce5..0x5a1d1c` | the unit's level (stat 0xc), at least 1, at most MonLvl's rows - 1 |
| `FUN_005a1c70` `0x5a1dcd` | cold length: stat 0x38 += level x 5 + 100 |
| `FUN_005a1080` | Ghostly's init: cold length 150 (unchanged here) |
| `FUN_005a00f0`, `FUN_005a0000`, `FUN_00611d30` | the MonUMod constants, the MonLvl row and the difficulty row the inits read (run for real by the oracle) |

Edge levels (verifier V run through game.exe's FUN_005a2120 with the base level forced to 0..200): every case equals 100 + 5 x min(max(final level, 1), 110), the final level including the unique's level add.

## Counted differentials (V run, the worker host, tree = upstream `62732598` plus the branch)

`umod_stats.py` runs game.exe's real `FUN_005a2120` and the mods' inits on a synthetic unit; `test_umod_stats --umod-stats` runs `rules::boss_stats` on the same cases the way `make_boss` applies it; `--compare` lists the fields that differ. cold_len mismatches (prep classifier `pp_cold_count.py`):

| Group | seed 1 (157,618 cases): parent | claim | seed 9 (107,618 cases): parent | claim |
|---|---|---|---|---|
| Cold Enchanted, no Fire / Lightning Enchanted | 7,869 | 0 | 4,745 | 0 |
| Cold together with Fire or Lightning Enchanted | 3,124 | 1,625 | 1,591 | 828 |
| minions of a Cold Enchanted boss | 5,657 | 5,657 | 3,732 | 3,732 |
| unlisted combinations (5 or more mods) | 2,290 | 248 | 1,154 | 135 |

(The verifier's own classifier counted 9,636 and 4,044 for the first row on r3 because it grouped the multi-mod rows differently; the claim-side 0 agrees.) The remaining rows are two earlier, separate deviations: `BossStats` keeps ONE element (`stats.elem`, the last Enchanted mod wins), and a minion has no mods in d2d (README row of `umod_stats.py`).

Glue test: `test_boss_cold_length` (real `make_boss`, 3 rows x 3 difficulties x Cold and Fire Enchanted) asserts the cold element's length is `100 + 5 * clamp(level, 1, 110)` and the fire element's 0. It fails on the parent and passes with the claim. `test_monsters` and `test_game`: rc 0.

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
- Overlap check (read-only `gh`, 2026-11-11): open #371 (unique mods 21, 35, 40-42, `uniques.hpp`) and #401 (`ai.cpp`) touch neighbouring files without touching `elem_len` or FUN_005a1c70 (verifier's `gh pr diff`).

## Checklist

- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same pull request (oracle commit first and building alone, claim second)
- [x] The finding in `docs/research/re/monsters.md`
- [x] `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN
- [ ] `python3 tools/presubmit.py --quick --base upstream/main` on r4: run it before posting (the r3 head: 0 FAIL except the `gcc` step on `apps/launcher/main.cpp`, `main.moc` missing in that build dir, a file this change does not touch)

Agent: Claude Code

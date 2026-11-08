# fid-inv-limits (TASK-79, matrix row IT10): purse cap, stash cap, grid sizes, d2d against game.exe

> status: finished 2026-11-08 by agent decomp5 (Sonnet). Result: caps and grids EQUAL; **one real d2d deviation found on the way (D1, the load clamp)** with a local PR candidate, card TASK-203. Nothing posted. Branch `local/fid-load-gold-clamp`, commit 42d40b09, pushed to the private fork remote the same day.

Tags: V read (disassembly, `tools/emu/d5_dis.sh` = upstream's disasm.py), V run (game.exe's own code run by the emulator, compared with d2d's real code in `test_game`), D (1.14d MPQ tables), I, U. d2d: upstream main 1453f853.

## Q1. Purse cap and stash cap in game.exe (AC #1)

| Fact | Evidence | Tag |
|---|---|---|
| Purse cap = clvl x 10,000: FUN_00622e70 = `FUN_00625480(unit, 12, 0) * 0x2710` (0x622e70..0x622e87) | disassembly | V read |
| Stash cap = 2,500,000: FUN_00623460 = `mov eax, 0x2625a0; ret 4` | disassembly | V read |
| The same two functions run natively on a real stat list for clvl 1..99 | `fl_gold.py limits`: **99 of 99 equal** to clvl x 10,000 and 2,500,000 | V run (`tools/emu/d5_gold_run.sh limits`) |
| The add helper FUN_0053ff00(unit, stat, delta) (20 call sites, e.g. in FUN_00540900 and FUN_0055c850 (ground gold), FUN_00564d50 (the stash command), FUN_00576d90 (pay), FUN_00566cc0 (death), FUN_00567c70, FUN_0055b060, and the transaction handler near 0x577e90; which are the sell path is not traced) sets stat 0xe or 0xf to **0** when the sum would pass the cap (0x53ff47..0x53ff6f), else adds | disassembly | V read |
| The sell handler FUN_00579510: the excess over the purse cap is returned to drop at the player's feet (FUN_0055b060) | d2d's own comment (rules.hpp `store_sell`); the handler was not run natively | I (from d2d's port), U for the native run |

d2d's sites: `rules::pay` (rules.hpp, pays through the FUN_0053ff00 zero rule), `store_sell` (`purse + price - over`), the ground pickup `room` (loot.hpp `take`: clvl x 10000 - gold), the stash deposit and withdraw (commands.cpp, `kStashGoldMax` 2,500,000 and `values[kLevel] * 10000`). All use clvl x 10,000 and 2,500,000: the numbers agree with game.exe's (V read of d2d, V run of game.exe).

## Q2. Grid sizes (AC #2)

From the 1.14d MPQ (D, `mpq-cat` of `data\global\excel\Inventory.txt` and `belts.txt`): every player class has inventory gridX 10 x gridY 4 (rows Amazon .. Assassin and the 800x600 `...2` rows); `Bank Page 1` 6 x 4 (classic), `Big Bank Page 1` 6 x 8 (expansion), `Transmogrify Box Page 1` 3 x 4 (cube), trade pages 10 x 4. Belt boxes (`numboxes`): belt 12, sash 8, default 4, girdle 16, light belt 8, heavy belt 12, uber belt 16.
d2d: the stash, cube and trade grids are read from the same file (`gamedata_load.cpp`, rows `Bank Page2`, `Big Bank Page2`, `Transmogrify Box2`, `Trade Page 1-2`, `Trade Page 2-2`); the player inventory is hard-coded 10 x 4 (rules.hpp, inventory.hpp: `free_spot(..., 10, 4, ...)`), equal to the table for all 7 classes (D). Belt sizes: the belt oracle of fid-belt-pr.md runs 390,000 cases with 8 belts on both sides with 0 differences (V run), so the box counts agree. Whether game.exe's own grid builder reads the same rows is V read only at the table level (not run): U.

## Q3. Does d2d clamp what a save holds? (found while tracing, D1)

game.exe's loader: FUN_0056b180 -> FUN_0056af80 puts the saved gold and stash gold back through FUN_0053ff00 (0x56afb4..0x56b00f, using FUN_00622e70 / FUN_00623460), so a save whose purse is above clvl x 10,000 or whose stash is above 2,500,000 enters the game with that value **at 0**. Oracle `tools/emu/fl_gold.py` (real FUN_0056b180 on a stat block with levels, gold and bank; same hooks as fl_load.py): 539 cases (11 levels x 7 gold values x 7 bank values: 0, 1, cap-1, cap, cap+1, 2 x cap, 2^25-1).

| d2d | cases | differ from game.exe |
|---|---|---|
| upstream 1453f853 (`World::take_character` keeps both) | 539 | **363** (231 purse, 231 bank) |
| local/fid-load-gold-clamp 42d40b09 (zero when over the cap) | 539 | **0** |

Reachability: a save from game.exe never exceeds the caps (levels do not drop). It needs an edited or imported save, or d2d writing one: U, so the severity is low and this is a fidelity candidate, not a bug that bites play.

## Deviation list

| # | d2d | game.exe | tag | differential |
|---|---|---|---|---|
| D1 | a loaded purse above clvl x 10,000 or stash above 2,500,000 is kept | set to 0 (FUN_0053ff00 via FUN_0056af80) | V run (363 of 539 before, 0 after) | yes, built: `fl_gold.py` + `test_game --load-gold` |
| D2 | none for the cap numbers or the grid sizes | | V run / D | |

## Bot findings (described, not changed)

B1 `internal/town/stash.go` (StashGoldCap 2,500,000): correct (V). B2 the pickit does not know the purse cap (price-trace.md B6, unchanged; gold above the cap stays on the ground on both sides). B3 the bot's inventory model uses 10 x 4 and stash 6 x 8 (expansion): equal to the table; not re-read here (U for the file:line).

## PR candidate

**Candidate, ready for the independent verifier: `local/fid-load-gold-clamp`, commit 42d40b09 (fork remote), one commit on upstream main 1453f853**: `components/game/world/setup.cpp` (+10), `tests/game/test_game.cpp` (+41: a unit block and the mode `--load-gold`), `tools/emu/fl_gold.py` (+87). `tools/pr_guard.py up/main HEAD`: 0 FAIL, 0 WARN; `presubmit.py --quick --base up/main`: 0 FAIL, 2 WARN (docs-sync, build skipped by --quick; to answer in the PR body); full `test_game` rc 0 with the new "OK: a save's gold over its cap is 0 on entering a game". Committed with the numeric noreply identity of the fork's earlier commits, last line `Agent: Claude Code`. Draft and Claims table: docs/research/pr-load-gold-draft.md (card TASK-203). The usefulness is low (U above); the decision to open it is the user's.

## Drill recipe

Lab binary with lab verbs (docs/forkdbg.md): `debug stat 14 <clvl x 10000 + 1>` on a throwaway character, save (not `--no-save`), re-enter: game.exe leaves the purse 0 (a cheat-made save is CHEATED for any other use, docs/probes/test-characters.md); on d2d before the fix it stays. Sell with the purse 1 gold under the cap and read the gold dropped at the feet (the U in Q1).

## Reproduce (worker host)

`tools/emu/d5_gold_run.sh limits` and `... load` (needs the tree built by `tools/emu/d5_gold_branch.sh`); the "before" count is the same script on a tree without the fix (`test_game --load-gold`).

# Draft (local, NOT posted): Zeal strike gap, FUN_00553b10 (SUPERSEDED upstream, one residual)

> status: 2026-11-12 (TASK-392, agent przeal). **The PR this card was to build is not needed: upstream main `276f85c0` (Dravis, merged 2026-11-12, "Zeal's next hit starts on the action frame, as FUN_005dbc60", test `tests/game/test_zeal_chain.cpp`) already implements the rule found by decomp13 (docs/research/pal-zeal-trace.md).** Branch `fid/zeal-gap-r1` was not pushed (no diff against upstream main `53910f55` would be left). What this card adds: the rule re-derived, a rate x weapon grid against game.exe on the parent and on current main, one measured residual (below), and a fork-only sandbox branch carrying the upstream commit. Nothing is posted or pushed upstream. Consumers: TASK-330 (zealdiff), the bot's kill-time model.

## The rule (V read, game.exe 1.14d)

| Address | What it does |
|---|---|
| `FUN_005dbc60` `0x5dbdd2..0x5dbe38` (srvdofunc 13) | after a strike: the remaining-hit counter (`FUN_006444a0` / `FUN_00644560`) is decremented; if it is still above 0 it finds the next target (`FUN_0056bd10`, filter `0x20003`) and calls `FUN_0056e210(unit, FUN_004efc20(skill))`; `FUN_004efc20` returns Skills record `+0x14c` (Param2); on the last hit nothing is called |
| `FUN_0056e210` | `FUN_00553b10(game = FUN_00554010(unit), unit, pct = Param2)` |
| `FUN_00553b10` `0x553b36..0x553c60` | clears the unit's events of types 0 and 1 (`FUN_00540e60`); new swing start tick = `now - (now - start) x (100 - pct) / 100`; re-adds one event per flagged AnimData frame (flags 1..4 from the record at unit `+0x50`, bytes from `+0x10`) at the tick the frame is reached at the unit's per-tick rate (unit `+0x4c`), then the end event |

With Param2 = 100 (Zeal and Fury, Skills.txt; Fend 60) the swing starts over at the strike. The next strike therefore lands `ceil(256 x action_frame / rate)` ticks later and the swing ends `ceil(256 x frames / rate)` after the strike: a function of the animation (frames, action frame, from AnimData) and the rate (attack speed with IAS and the weapon speed modifier), not a constant 7. d2d before `276f85c0` waited out the whole swing (`pmode_until`) between strikes.

## Numbers (V run, the worker host; harness `tools/emu/zg_*`, game side FUN_00553b10 natively, d2d side the real `Fight` of each tree)

Grid: 5 weapon codes x IAS {0, 20, 40, 60, 100, 150} x weapon speed {-30, -20, 0, 10, 20}, Zeal level 4 (five strikes, four gaps per cell), 150 cells, 600 gaps. Raw lines: `docs/upstream/zeal-gap/grid_before.txt`, `grid_after.txt`.

| d2d tree | cells that differ | gaps that differ | d2d gap minus game gap |
|---|---|---|---|
| parent of `276f85c0` (`b6b59ae9` line, the sandbox lineage) | 150 of 150 | 600 of 600 | +4 to +10 ticks (e.g. 15 against 7 at rate 256; a 5-strike cast 75 against 43 ticks) |
| upstream main `53910f55` | 50 of 150 | 200 of 600 | 0 on 400 gaps, **+1 tick on 200** |

## The residual (V run, cause V read, game side for the FIRST strike U)

The 50 cells are those where the swing length in whole ticks is shorter than the full frame count at rate 256, e.g. rate 307: d2d strikes 7 ticks after the restart, game.exe's rollback gives 6. Cause: `Fight::set_pmode` (`setup.cpp:445-469`) takes `prate = animation length / (attack_ticks x 40 ms)` and the action time as `action_ms / prate`; that scales the action frame by the rounded-up swing length (13 of 15 ticks at rate 307: 242.7 ms, 7 ticks) where the game's event tick is `ceil(256 x 7 / 307) = 6`. This is the same rule that places a plain attack's FIRST strike (d2d's first strike is also 7 at rate 307), so a fix would not be Zeal-specific: it changes every attack's action tick. Whether game.exe puts its first swing's action event at `ceil(256 x action / rate)` too is NOT shown here (the harness only runs the rollback), so this is not a PR candidate; it needs an oracle on the first-swing event (`FUN_0062a710` / the unit's first events) first. Effect size: one tick in 6 to 9 at the common rates, 0 at rate 256 and 204.

## Limits

- The weapon argument of `zg_zeal_test` did not change the swing animation: all five codes (sword, dagger, great sword, spear, staff) printed the same 15-frame, action-frame-7 record (`A 15 280`), so the grid exercises ONE animation record (PAA11HS class) at 30 rate pairs, not several weapon classes. The other records' numbers (17/8, 18/8, 20/8, 18/9, 14/7) are decomp10's game-side runs only. Fixing the harness (equipping through the item path that sets the hit class) is the next step. U for those classes on the d2d side.
- Frames and action frame are read from d2d's AnimData (the same file as game.exe's); the unit-field meaning of the game-side harness is I (the end event equals `ceil(256 x frames / rate)` in all 150 cells: that is the check).
- Skill level 4 only; Zeal's mana is paid once per cast on both sides (`M 3840 2304`).
- Dragon Talon kicks and Fend use `kicks_left` in `Fight::player_modes`; upstream's change only restarts kRepeatedHit (Zeal, Fury). Fend's Param2 is 60 in the commit's note; not tested here.

## Fork-only sandbox

Branch `fork-only/sandbox-zeal` on the private fork remote: `a42dba25` (sandbox-itemverbs, the lineage of image `d2dbot:iv1`) plus upstream `276f85c0` cherry-picked as `44ed27f6` and `90ac00ab` (head; the test reads the action time from the animation, this lineage has no `action_ms` member; `test_zeal_chain` and `test_game` rc 0 on the worker host) (conflict in `strike.cpp`: this lineage has no `start_delay` call after the strike; resolved by keeping the lineage's lines). The image was NOT swapped.

## Re-run

```sh
tools/emu/zg_tree.sh WORKTREE BUILD_DIR OUT_BINARY      # builds libgame of the tree, compiles zg_zeal_test
tools/emu/run-on-worker.sh --ttl 1500 zg_run.sh OUT_BINARY [LEVEL]   # the grid; exit 1 when a gap differs
```

Agent: Claude Code

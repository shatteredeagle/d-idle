# fid-warp-arrival (TASK-350, consumer passagebounce TASK-349): where a warp puts the player, and what arms the return exit

> status: finished 2026-11-11 by agent decomp12 (Sonnet), offline. Result: **one proven d2d deviation (the arrival spot search), fix candidate pushed to the private fork; the arrival bounce itself is NOT explained** (the click-box rule is the same in d2d and game.exe, and it hits about 18 percent of clicks, not 88 percent: U, drill below). Nothing posted, no live game, no sandbox touched.

Tags: V run (game.exe's own functions in the emulator against the real compiled d2d, worker host), V read, S (an existing doc), D, I, U. d2d: upstream main 8d7b89f1 (scratch tree `wt-decomp12`, build `build-decomp12`; `tools/emu/d12_setup.sh`, `d12_build.sh`, `d12_run.sh`, `d12_fixrun.sh`, probe `d12_arrive_probe.cpp`). Seeds 5, 82, 7 for the level windows.

## Q1. Where does game.exe put the player after a level warp?

FUN_005550b0 (S, drlg.md "Taking a warp", `docs/research/re/drlg.md`): the other side's tile unit of the linked class (FUN_006195a0), then the free spot nearest that unit (FUN_00554ea0 -> FUN_0064e7b0 -> FUN_0064dea0, player pattern, mask 0x1c09), then a server walk (FUN_005809d0) to that spot + the LvlWarp row's ExitWalkX/Y (subtiles). The unit stands at the tile's cell x 5 + OffsetX/Y (FUN_0066e1c0). The spot rule: the unit's subtile if free, else the free subtile of least |dx| + |dy| in the first square ring that has one, rings 1 to 49, ties in FUN_0064dea0's scan order.

d2d `World::use_warp` (travel.cpp) takes the same unit (`Level::Warp::unit_x/unit_y`, (cell x 5 + Offset + 0.5) / 5), then `World::arrive` calls `Level::nearest_free`, then sets `walking` toward spot + ExitWalk.

**Spot search, game.exe against d2d, V run.** `vb2_tele.py` runs FUN_0064dea0 (the arguments FUN_0064e7b0 passes for a warp) natively on a 200 x 200 subtile grid (the plus footprint hooked, S); `d12_arrive_probe synth` runs d2d's REAL `Level::nearest_free` (as `arrive` calls it, from the centre of the unit subtile) and `Level::warp_spot` on the same cases (a copy of a real level with the grid cut into it).

| seed 7, 20,000 cases (19,574 land in game.exe, 18,952 move off the asked subtile) | differ |
|---|---|
| d2d `nearest_free` (what `arrive` uses) | **10,265**: 9,839 land on the same ring but another subtile (ring scan order, no least |dx|+|dy| rule); 426 game.exe refuses (no free spot within ring 49) and d2d lands on ring 50/51 |
| d2d `warp_spot` (proven for Teleport) | **0** |

**On the real Act 1 levels (V run).** `d12_arrive_probe levels SEED` takes every warp of every level d2d builds for the seed (52 arrivals), cuts a 200 x 200 window of the DESTINATION level's collision round the return warp's unit (d2d's collision, proven equal to game.exe's on 100 of 131 levels, S; the spot search itself is game.exe's), and compares:

| seed | arrivals | unit subtile blocked (hero is moved) | `nearest_free` differs from game.exe | `warp_spot` differs |
|---|---|---|---|---|
| 5 | 52 | 28 | **26** (all ring 1) | 0 |
| 82 | 52 | 28 | **26** | 0 |
| 7 | 52 | 28 | **25** | 0 |

So d2d lands one subtile off game.exe on about every second warp whose unit subtile is blocked (those 28 of 52 are mostly the way back out of a cave: Passage to Stony Field and Dark Wood, Crypt to Burial Grounds, the Den and the Cave levels). The other 24 arrivals have a FREE unit subtile, among them Blood Moor to Den, Cold Plains to Cave 1, Stony Field and Dark Wood to the Passage: both put the hero exactly on the unit (offset 0,0), no deviation there.

## Q2. When can the warp trigger fire?

Server (S, collision.md and drlg.md, spot checks V read by me): a warp is taken only by C->S 0x13 on the tile unit (FUN_00548b00 case 5): distance FUN_00641530 under 5 takes it at once, under 0x33 the player walks toward it, else nothing. A plain walk (0x03) never takes a warp. There is no "last warp" flag or arrival grace period in that path (nothing read; absence not proven).

Client (V read): a click becomes a warp click only when the cursor is in the hit box FUN_00463ee0 (FUN_00466870 case 5): the tile unit's LvlWarp row (SelectX/Y/DX/DY), unit subtile in world pixels ((x - y) x 16, (x + y) x 8), edges in. FUN_00461dc0 type 5: refused until the time at `0x7a04c8`, which FUN_0044dad0 sets to now + 500 ms. Callers of FUN_0044dad0 (V read, whole-binary call list): 0x462022 (after a warp click), 0x48168f, 0x46706b, 0x46708c; **none is a level-change handler**, so the 500 ms is a click throttle after a warp click, not an arrival grace. The hit test also needs unit flags (+0xc4 bit 1/21, +0xc8 bit 7 at 0x466974 to 0x466986): not decoded (U).

d2d (V read, `World::walk_to`, commands.cpp:616): a FRESH `cmd::Move` (also the devctl `cmd move`, the bot's only walk verb) sets `take_warp` for every warp whose Select box holds the click cell and retargets the unit; `use_warp` runs every tick and takes it when `warp_close` (< 5). `arrive` clears `take_warp`. Same box formula as game.exe (V read both, upstream `test_warp_click.cpp`). Differences to game.exe: no 500 ms refusal, no distance-0x32 limit, tested every tick (collision.md, S).

## Q3. Why do 88 percent of Passage arrivals bounce? NOT explained (U)

What the data say (V run, d2d's own `World::warp_clicked` / `warp_close` at the arrival spot, 16 directions x distances 2, 4, 8, 12, 16, 20 subtiles = 96 clicks from the hero):

| arrival | unit free | hero on unit, reach (<5) | clicks inside the return warp's box at arrival | after ExitWalk (3,5): reach / clicks inside |
|---|---|---|---|---|
| Stony Field -> Passage (4 -> 10) | yes | yes | 17 of 96 (18 %) | no / 2 of 96 |
| Dark Wood -> Passage (5 -> 10) | yes | yes | 17 of 96 | no / 2 |
| Burial Grounds -> Crypt (17 -> 18) | no (moved 1 subtile) | yes | 7 of 96 | no / 2 |
| Blood Moor -> Den (2 -> 8), Cold Plains -> Cave 1 (3 -> 9) | yes | yes | 17 of 96 | no / 2 |
| Passage -> Stony Field (10 -> 4), Passage -> Dark Wood (10 -> 5) | no | yes | 13 and 7 of 96 | no / 2 |

So with uniformly spread click directions only about 7 to 18 percent of first orders would hit the box, and after the ExitWalk finishes only 2 to 3 percent; the sandbox saw 88 to 100 percent. Either the bot's first orders are not uniform (the roam target often lies in the direction the box points, the cave body; the bot's step-away `mouthStep` moves along the exit position, not the box), or another d2d path arms the return that I did not find (nothing else writes `take_warp`: grep of travel.cpp, commands.cpp, tick.cpp, walk.cpp). The hero at arrival IS within reach of the return unit in every case, so a single in-box fresh Move fires the exit on the next tick (0.2 s with the command latency fits). **U**: which click cells the bot sent (the passagebounce `arrival bounce` lines carry arrival and leave positions, not the click), and whether the box side of the sandbox arrivals was the cave body.

Dead ends: a "just warped" flag in game.exe (the only timer near, `0x7a04c8`, is a click throttle); a d2d arrival grace (none; `arrive` leaves `walking` set toward the ExitWalk target and a fresh click cancels it, as in game.exe).

## Deviations

| id | game.exe | d2d | tag | differential | PR |
|---|---|---|---|---|---|
| D1 | spot = unit subtile if free, else least |dx|+|dy| in the first ring, rings 1..49 (FUN_0064dea0) | `Level::nearest_free`: first free in ring scan order, rings up to 199 | V run (20,000 synthetic: 10,265 differ; real Act 1 windows: 25 to 26 of 52 differ per seed) | yes, built (`d12_run.sh synth|levels`) | see PR candidate |
| D2 | warp click refused for 500 ms after a warp click (FUN_0044dad0, 0x7a04c8) | none | V read | buildable: FUN_00461dc0 type 5 with a faked clock, oracle: whether the packet is sent | no |
| D3 | warp click walks and queues 0x13 when farther than 4; tested when the packet arrives and when the walk ends | tested every tick, takes the warp on the way up to 4 subtiles early | S (collision.md) | not built | no |
| D4 | click box needs unit flags (+0xc8 bit 7) | none | V read, flags U | not built | no |

## Bot findings (described, not changed)

B1: `internal/nav/walk.go:777` (`mouthStep`) steps along the line from the listed exit position; the trigger is the Select box from the tile unit, which reaches up to 15 subtiles up-screen from it (e.g. Cave Up SelectY -120 = dx+dy down to -15), so "straight away from the listed position" can still be inside the box on one side. B2: the walk radii (`mouthR` 9, `mouthArrivalR` 16, `mouthApproachR`) are fits to a box, not the box; the box per warp is `LvlWarp` data (`debug warps` has only a point). B3: the passagebounce breaker (3 bounces in 2 minutes bans the area) keys on area and is the right guard for any cause: **keep it** after D1 lands, because D1 only moves the hero by one subtile in the blocked-unit cases and does not touch the main free-unit Passage arrivals.

## PR candidate (private fork only, nothing upstream)

Branch `fid/warp-arrival-r1`, commit **feba9539**, pushed to the private forge remote the same day (base upstream 8d7b89f1): `World::warp_arrival` (Level::warp_spot, the proven FUN_0064e7b0 rule), used by `use_warp`; waypoints, portals and quest arrivals untouched. 12 added lines in `components/game/world.hpp` and `world/travel.cpp`.

Before: 25 to 26 of 52 real arrivals and 10,265 of 20,000 synthetic cases differ. After: the probe built against the patched tree calls `World::warp_arrival` itself: **0 of 52 differ on seeds 5, 82 and 7** (`d12_fixrun.sh`). The synthetic run of the helper is the `warp_spot` row (0 of 20,000), since the helper is `warp_spot` plus a fallback. Not run: the full `use_warp` through a `World` (the probe calls the helper, not the glue), the d2d unit tests on the patched tree (U). **It does not stop the bounce** (Q3). Independent verifier (d2-reviewer) must check it before any upstream step; no upstream text drafted.

Sandbox rebuild steps: on the worker host the scratch worktree wt-decomp12fix under the lab directory (`git log -1`) is feba9539; `cmake --build build-decomp12fix -j 6 --target d2d` in the lab directory (the fork's d2d target; `d12_fixrun.sh` built only `game`), then copy the binary as the lab docs (docs/d2d-fork.md) describe. The bounce will still be there.

## Drill recipe (lab, nothing run here)

1. `debug warps` right after an arrival in the Passage, then `cmd move` to 16 cells around the hero at 2, 8 and 20 subtiles (the `mouth.py` method of docs/d2d-loop.md, r18): record which fire; compare with the table in Q3 (box = LvlWarp row 4 for Cave Up, 5 for Cave Down: x -30..90, y -120..30 and x -25..55, y -75..25 from the unit pixel).
2. Log the click cell of the first bot order after an arrival (passagebounce `arrival` line plus the order) and test it with `World::warp_clicked` (the probe's formula): if it is in the box the cause is the bot's click choice; if not, something else arms the exit: `debug level` before and after the order.
3. With the fix binary, repeat 1 on the blocked-unit arrivals (Crypt, Passage back) and check the hero's subtile equals the probe's C column.
4. game.exe side (needs the user's go, not run): a lab game, arrive by a cave mouth, click 2 to 8 subtiles away in the box direction, see whether it re-enters.

## Matrix-cell updates (proposed text)

For the warp/arrival row of the fidelity matrix: "arrival spot: D1 (ring order), fix candidate fid/warp-arrival-r1 feba9539; click box: same as game.exe (V read both); 500 ms click throttle missing (D2, V read); arrival bounce cause U (decomp12, fid-warp-arrival.md Q3)".

## Not done

Real-level game.exe arrival through the whole server chain (FUN_005550b0 needs the room/unit lists; only the spot rule was run natively on d2d's proven collision); decoding the hit-test unit flags; any live or sandbox observation.

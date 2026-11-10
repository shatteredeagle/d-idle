# fid-wp-tp-pr (TASK-95, matrix rows PR06 PR07): waypoint and town portal, d2d against game.exe

> status: finished 2026-11-09 by agent decomp8 (Sonnet). Result: **both oracles are 0 differ on current upstream main 83fc7ba1 against the real build**: the waypoint refusal window (2 x 140,002 cases) and the Town Portal read from the pack (2 x 108,000 cases). The deviations the 2026-10-31 traces found (waypoint #153, town portal: cast instead of instant, busy drop, charge on a cut cast) are gone from main. AC #1 and the oracle half of AC #2 proven; three sub-items of AC #2 (portal 0.6 cells south, expiry, scroll count past the pack) and AC #3 are not run (U, below). Nothing posted, no PR candidate.

Tags: V run (game.exe's own functions in the emulator against the real compiled d2d, worker host), V read, S (an existing doc), D, I, U. d2d: upstream main 83fc7ba1 (scratch tree and build `build-decomp8`, `tools/emu/d8_setup.sh`, `d8_build.sh`). Scripts: `tools/emu/d8_wp.sh`, `tools/emu/d8_tome.sh`.

## Q1. Waypoint refusal window (C->S 0x49), game.exe against `World::apply(cmd::Waypoint)`: 0 differ, V run

`wp_refuse.py gen 100000 {7,271828} --solo` (grid of 0..20,000 ms since the last level change x target active, 40,002 cases, plus 100,000 random single-player timelines with 1 to 3 level changes) = 140,002 cases per seed. game.exe side: `wp_refuse.py game` (FUN_0054d750 -> FUN_0054c5d0 -> FUN_0055b6c0 / FUN_00554190 native). d2d side: `test_game --waypoint-refuse` (real `World::arrive` level changes, real `World::apply`).

| seed | cases | game lines | d2d lines | differ | md5 of both outputs |
|---|---|---|---|---|---|
| 7 | 140,002 | 140,002 | 140,002 | **0** | e38fb1425e1ed7093048233c31e56bd7 (both; the same value as the verified run of the PR #153 proof) |
| 271828 | 140,002 | 140,002 | 140,002 | **0** | d154b1f9d92c392c4714524341cd8a72 (both) |

So the claim of pr-wp-proof.md (game.exe refuses only after a hostile declaration, never in single player; d2d refused for 10 s after every arrival) holds on main: fixed by PR #153 (merged 2026-11-01), still equal 8 days and 100+ commits later. The `--solo` limit stands: hostile declarations (PvP) are not modelled by d2d, so the cases with them (30,763 of 132,964) are out of scope (S, pr-wp-evidence-pack.md).

Not covered by this oracle (honest): the target check of FUN_00549570 (range 10 subtiles, 22 for a Sorceress by Telekinesis, I; same act; waypoint index; known-waypoints bit via FUN_00660e00) and the dispatcher's dead/busy drop. The pack listed them as "existing d2d gaps"; I did not re-read d2d's `commands.cpp` for them on 83fc7ba1 (U). The known-waypoints block lifecycle has its own stock-only check (`waypoint_lifecycle.py`, "checks stock behavior, not d2d equivalence": not a differential).

## Q2. Town portal read from the pack (C->S 0x20): 0 differ, V run

`tome_use.py gen 100000 {7,271828}` (tbk/tsc x charges -2..20, 255, 2^20 x town x item on cursor x player mode 0..19 x free spot, then random incl. state 54) = 108,000 cases per seed. game.exe: FUN_0054d750 -> FUN_0054b1e0 -> FUN_0055e170 -> FUN_005bf240 -> FUN_005be290 native (hooks in the docstring of `tome_use.py`). d2d: `test_game --tome-use` (real `World::read_portal`).

| seed | cases | game lines | d2d lines | differ |
|---|---|---|---|---|
| 7 | 108,000 | 108,000 | 108,000 | **0** |
| 271828 | 108,000 | 108,000 | 108,000 | **0** |

Before (town-portal-trace.md section 2, d2d f8aab0e8, a line copy of the old code): 3,109 cases "game.exe opens at once, d2d at the SC frame", 7,930 "game.exe opens, d2d drops the click (busy mode)", 3,116 + 7,962 no free spot (charge spent, old pair kept), 278 state 54. All four differences are gone on 83fc7ba1: `World::read_portal` (components/game/world/travel.cpp:285) now answers like game.exe on every case (shown by the run; its body was not re-read, and the commit that changed it between f8aab0e8 and 83fc7ba1 was not bisected, U). Position, twin and one pair per player were not compared by this oracle: the free spot is a hook (per case), so the portal position rule (size 3, mask 0x3e01, FUN_0064e810 from the caster) is S only (town-portal-trace.md).

## Q3. What is not run (open sub-items of AC #2)

| item | state | tag |
|---|---|---|
| Portal "0.6 cells south" (the live observation of the old matrix note) | not a game.exe rule found; FUN_0064e810 searches the nearest free spot from the caster, so an offset is the search result, not a constant | I |
| Portal expiry | no timer seen in the chain of FUN_005be290 / FUN_0056d130 / object 59 (town-portal-trace.md): it lasts until replaced, used back by its owner, or the game ends; the 5 s of FUN_00584870 is the hostility window (same field as the waypoint), not an expiry | V read (old), U for a timer in the object's tick |
| Scroll count past the pack, scroll vs tome precedence of the skill-bar path (skill 219/220, FUN_005bf3d0) | the oracle covers the inventory path (0x20) only; the skill-bar path is not compared | U |

## Q4. Shim R06 (act 3-5 waypoint coordinates), AC #3

R06 (docs/d2d-deviations.md) is a bot shim: d2dworld reports act-absolute positions for acts 3 to 5, the bot adds the level origin from `internal/nav/act2hints.go:knownWaypointRel` and `internal/nav/waypoint.go:knownWaypointRel`. game.exe has nothing to compare (the origin is the client's level rect); the removal condition stays as the register states it ("move the table into the adapter or delete it when d2dworld reports level-relative positions"). Not re-verified on main: whether d2d's state reader still reports act-absolute positions (U). Not run: needs the lab (recipe below).

## Deviations (d2d against game.exe)

None in the compared rules (0 of 280,004 waypoint cases and 0 of 216,000 tome cases).

| id | what | tag | differential |
|---|---|---|---|
| D1 | hostile (PvP) refusal window | V run earlier (30,763 of 132,964 cases differ, S) | not needed: d2d has no PvP; could be built by modelling declarations |
| D2 | waypoint target check (range, same act) and dispatcher dead/busy drop | U (pack called them existing gaps) | could be built: extend `wp_refuse.py` to hook FUN_00549570 un-hooked with unit positions per case and `test_game --waypoint-refuse` with a player in range / out of range, oracle: game.exe's result code 0..3 |
| D3 | portal position and twin | S only | could be built: run FUN_0056d130 un-hooked with the room's collision, compare `portal_spot`; about 150 lines of C++ plus an emu script |

## Bot findings

B1: `WpCooldownSec` should be 0 now (pack, "bot follow-up after the merge"): not re-read here for the file:line (U). B2: the soak lines of town-portal-trace.md section 4 (empty tome picked first; click while busy) stay valid for the bot but the busy half is moot on d2d main: a click while in a get-hit mode now opens a portal at once as game.exe does. No code changed.

## PR candidate

No PR candidate: nothing deviates (0 of 496,004 cases).

## Drill recipe (lab, not run)

- Waypoint right after arrival: lab binary (docs/d2d-fork.md), throwaway Sorceress, stand on a waypoint in Blood Moor after a level change, send the 0x49 within 2 s (`debug` verbs of docs/forkdbg.md); expected travels at once.
- Town portal in a get-hit mode: a charged tome, take a hit, click the tome: expected portal at the click (mode GH), charge minus one.
- R06: `debug state` in Act 3 town, read the waypoint object position, compare with `knownWaypointRel` + level origin.

## Matrix-cell update (proposed)

PR06: "UPDATE 2026-11-09 (decomp8, fid-wp-tp-pr.md): refusal window 0 differ on 83fc7ba1, 2 x 140,002 cases, real World::apply (V run); target check (range, act) not compared (U)."
PR07: "UPDATE 2026-11-09 (decomp8): Town Portal read from the pack 0 differ, 2 x 108,000 cases (V run), the four old deviations are gone; position/twin and the skill-bar path not compared; no expiry timer seen (V read, old)."

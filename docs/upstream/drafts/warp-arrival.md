# Draft (local, NOT posted): "A warp lands on the free subtile nearest its unit, as FUN_005550b0"

> status: r2, ready for the second verifier pass (prwarp, TASK-372, 2026-11-12). Branch `fid/warp-arrival-r2` on the private fork remote, head `HEAD_HASH`, two commits on upstream main `c9b4c06e`: oracle `ORACLE_HASH`, claim `CLAIM_HASH`. Supersedes `fid/warp-arrival-r1` (feba9539, stale on the remote, never force-pushed). First verifier (TASK-366): [../verification-warp-arrival-2026-11-12.md](../verification-warp-arrival-2026-11-12.md), NEEDS-WORK on r1; every edit it asked for is applied here (list at the end). Claim doc: [../../probes/fid/fid-warp-arrival.md](../../probes/fid/fid-warp-arrival.md).

Nothing here is posted. PR title: `A warp lands on the free subtile nearest its unit, as FUN_005550b0`

## What changes

`World::use_warp` put the player down with `Level::nearest_free`, which scans each square ring in its own order (`i,-ring`, `i,ring`, `-ring,i`, `ring,i`) and gives up at ring 199. When the other side's warp unit stands on a blocked subtile, game.exe takes the free subtile of least |dx| + |dy| in the first ring that has one, rings 1 to 49, ties in FUN_0064dea0's scan order; d2d landed on another subtile of the same ring about every second such arrival (the way back out of a cave, the Crypt, the Den).

`World::warp_arrival` returns `Level::warp_spot` (that rule; already used for Teleport, `fight/strike.cpp`) as a cell position, and `use_warp` lands through it. game.exe searches twice: FUN_005550b0 calls FUN_0064e7b0(room, &xy, size 2, mask 0x1c09, flag 1) (`0x555144..0x555157`), then FUN_00554ea0 from that spot (`0x5551bc`, flag 0). When the flag-0 search finds nothing the warp is not taken (`0x5551c3`), so `warp_arrival` returns `std::optional` and `use_warp` logs and returns: the hero stays where he is and `take_warp` is already cleared. Waypoints, portals and quest arrivals keep `nearest_free`. Code: 11 added lines in `components/game/world.hpp`, 5 in `world/travel.cpp`.

Two commits: the oracle (`tools/emu/sk_tele_spot.py warp` and `gen --warp`, d2d side `tests/game/test_warp_arrival.cpp --warp-arrival`; builds on its own and prints what `use_warp` did before, so it shows the difference on the parent), then the claim (`world.hpp`, `travel.cpp`, the test's hand cases and the whole-`use_warp` part, `docs/research/re/drlg.md`).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005550b0` `0x555144..0x555157` | `FUN_0064e7b0(room, &xy, FUN_00620510(player) = 2, 0x1c09, 1)` from the tile unit's subtile; 0 returned ends the function (`0x555160`) |
| `FUN_005550b0` `0x5551bc..0x5551c3` | `FUN_00554ea0(room, x, y, 0, 0)` from that spot; 0 returned: no warp |
| `FUN_00554ea0` `0x554f30..0x554f43` | with its fourth argument 0 it calls `FUN_0064e7b0(.., 0x1c09, 0)` again, then `FUN_00650be0` places the unit |
| `FUN_0064e7b0` | `FUN_0064dea0(.., 0x32, 1)` |
| `FUN_0064dea0` | the start subtile when it is in a room and free, else rings r = 1.. while r + 1 < 0x32 (`0x64e1e5..0x64e1f8`): the columns x - r and x + r row by row, then the rows y - r and y + r column by column; a later free subtile wins only when |dx| + |dy| is strictly less (`jge`, `0x64e088` and `0x64e181`); the first ring with a free subtile ends the search; none in 49 rings: flag set returns the start unchanged, flag clear returns 0 |
| `FUN_0064d9b0` size 2 | the plus footprint (the subtile and its four neighbours) against mask 0x1c09 |

## Counted differentials (V run, the worker host, tree = upstream `c9b4c06e` plus the branch)

Oracle, synthetic: `sk_tele_spot.py warp` runs game.exe's real FUN_0064e7b0 and FUN_00554ea0 (including FUN_0064dea0 and FUN_0064d9b0) on a 200 x 200 window with walls, doors and objects round the unit (5 percent of the cases fill every ring to 44..50 so that some are refused); `test_warp_arrival --warp-arrival` prints the real `World::warp_arrival` on the same cases.

| 20,000 cases (units' bit left out) | differ, parent (`Level::nearest_free`) | differ, claim |
|---|---|---|
| seed 1 (19,692 land in game.exe, 308 refused) | **11,645** (11,337 land on another subtile, 308 d2d lands where game.exe refuses) | **0** |
| seed 31 (19,716 land, 284 refused) | **11,249** (10,965 and 284) | **0** |

Real levels, the whole path: `pw_usewarp` builds a `World` for a seed, puts the player on each warp's unit of every level, sets `take_warp`, calls `World::use_warp()` (rooms, objects, doors and `arrive`'s second `nearest_free` included) and compares the hero's final subtile with FUN_0064dea0 run natively on a 200 x 200 window of the destination's collision (taken after the arrival; the pre-arrival window gives the same in Act 1). 28 of the 52 Act 1 arrivals per seed have a blocked unit subtile (the hero is moved):

| Act 1 (levels 1..39, 52 arrivals a seed) | parent | claim |
|---|---|---|
| seed 5 | 26 | **0** |
| seed 82 | 26 | **0** |
| seed 7 | 25 | **0** |
| seed 11 | 25 | **0** |
| seed 23 | 26 | **0** |
| seed 4242 | 25 | **0** |

Acts 2 to 5 (levels 40..136, seed 5, 140 arrivals, 113 move the hero off the unit): parent **92** differ, claim **2** (post-arrival window; 3 on the pre-arrival window), see the limits.

Glue test: `test_warp_arrival` (no argument) asserts ten hand cases (answers from the same oracle: a free unit, a blocked one, the corner order, ring 48 and 47, refusal at ring 49), and with the game data runs `use_warp` on seed 5 through three warps whose unit is blocked (Passage to Dark Wood, two more) and asserts the hero's subtile equals game.exe's answer; then blocks the whole destination and asserts the warp is refused (hero on the same level, same position, `take_warp` cleared). It SKIPs the data part without the MPQs and the 1.14d installer.

## How to re-check

```sh
python3 tools/pr_guard.py upstream/main HEAD
cmake --build build --target test_warp_arrival
cd tools/emu && uv sync
set -o pipefail
uv run python sk_tele_spot.py gen --warp 20000 1 > /tmp/cases.tsv
uv run python sk_tele_spot.py warp /tmp/cases.tsv > /tmp/game.tsv
../../build/tests/test_warp_arrival --warp-arrival /tmp/cases.tsv | grep -E '^[0-9]+ ' > /tmp/d2d.tsv
uv run python sk_tele_spot.py diff /tmp/game.tsv /tmp/d2d.tsv /tmp/cases.tsv
../../build/tests/test_warp_arrival
```

## Honest limits (to put in the body)

- The oracle's world is one room for the whole 200 x 200 window, as d2d's collision is one array. FUN_0064dea0 looks each subtile's room up from the room it last found, so on a level of several rooms a ring can lose a neighbour room; that is not modelled (U). The real-level runs above use d2d's collision of real levels with the spot rule run natively; d2d's collision equals game.exe's on 100 of 131 levels (S, `fid-warp-arrival.md`). The footprint test at a room edge (`FUN_0064d100`'s edge cases) was not read.
- Units: `warp_spot` tests the grid without the units' bit (mask 0x0c09; game.exe passes 0x1c09). A monster standing on the exit subtile moves game.exe's hero and not d2d's. This is the same for Teleport (pre-existing; d2d keeps units in the Crowd, not the grid).
- Acts 2 to 5, seed 5: two of 140 arrivals still differ, both outside this change. (1) Kurast Sewers level 1 to 2: d2d keeps the stairs shut until the lever (`use_warp`'s quest gate, unchanged here), so it does not arrive; game.exe's FUN_005550b0 gates the same level with FUN_00545b80. (2) Level 51 to 52: an object made on arrival blocks the spot, and `World::arrive` re-places the hero with `nearest_free` after `init_new_objects` / `reopen_doors` ("off what the room just made"), whose ring order differs again; 1 of 139 reached arrivals, none in Act 1 (0 of 312). A follow-up can route that second re-place through `warp_spot` too; it is a separate change in `arrive`.
- No real level reaches the refusal (game.exe found a spot on all 452 real arrivals of both trees); it shows in 308 / 284 of the 20,000 synthetic cases. The refusal is by FUN_00554ea0 reading, checked by the oracle, not by a level.
- This does not stop the Passage arrival bounce (`fid-warp-arrival.md` Q3, a separate open question); it only moves the hero by one subtile in the blocked-unit cases.
- Overlap (read-only `gh`, 2026-11-12): no open PR or issue covers warp arrival; #433 touches `world.hpp` (an event struct and a `Player` field, far from `arrive`).

## Checklist

- [x] A test in `tests/` citing the game.exe addresses, and the oracle in `tools/emu/`, in the same pull request (oracle commit first and building alone, claim second)
- [x] The finding in `docs/research/re/drlg.md` ("Taking a warp", and the table row now "diffed")
- [x] `python3 tools/pr_guard.py upstream/main HEAD`: PRGUARD_LINE
- [x] `python3 tools/presubmit.py --quick --base upstream/main`: PRESUBMIT_LINE
- [x] `ctest -j8` on the full build: CTEST_LINE

---

Agent: Claude Code

## Verifier edits applied (r1 to r2)

| # | Verifier finding | Where |
|---|---|---|
| 1 | no test | `tests/game/test_warp_arrival.cpp` (hand cases, whole `use_warp` on seed 5, refusal) |
| 2 | no oracle in the PR | oracle commit first: `sk_tele_spot.py warp`, `gen --warp`, README row |
| 3 | research doc | `drlg.md`: "Taking a warp" (two searches, ring rule, refusal), table row "diffed" |
| 4 | fallback contradicts the claim | `warp_arrival` returns `std::optional`; `use_warp` logs and returns (tested) |
| 5 | wrong caller cited | comment cites FUN_005550b0 -> FUN_0064e7b0 (`0x555144..0x555157`), FUN_00554ea0 re-checks |
| 6 | commit format | new branch off `c9b4c06e`, oracle first, short subjects, three body lines, `Agent: Claude Code`, the user's own identity |
| 7, 8 | disclose | two lines in the limits above (arrive's second `nearest_free`, units' bit) |

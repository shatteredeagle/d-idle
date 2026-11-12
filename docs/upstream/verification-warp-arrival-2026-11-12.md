# Independent verification of the warp-arrival fork candidate (2026-11-12)

> status: current, 2026-11-12 (TASK-366, agent prverify3). Verdict **NEEDS-WORK**: the rule and the code change are right and proven; the branch is not upstream-ready (no test, no oracle in the PR, no research-doc row, commit message format). Nothing posted. Consumer: TASK-45 (upstream PR posting).

Verifier: d2-reviewer (Opus), singly. Candidate: `fid/warp-arrival-r1`, commit `feba9539` on the private fork remote (one commit on upstream `8d7b89f1`), claim doc `docs/probes/fid/fid-warp-arrival.md` (TASK-350). Standard: `docs/upstream/verification-2026-11-11.md` and memory d2-pr-proof-standard. Tags: V read (disassembly read here with upstream `tools/emu/disasm.py`), V run (run here), S, U.

## Setup (V run)
- Upstream main is now `c9b4c06e` (235 commits after the candidate's base). The commit cherry-picks with no conflict (`f0e8b396`, local only). Nothing upstream since `8d7b89f1` touches `nearest_free`, `warp_spot` or the `arrive` call in `use_warp` (`git log -S`); the travel.cpp commits in the range are #448's walking gate and the quest-clock series.
- Open upstream PRs on the same files: #433 edits `world.hpp` (an event struct and a `Player` field, far from `arrive`): textual only. No upstream PR or issue covers warp arrival (`gh search prs` "warp arrival", "nearest_free"; read only).
- Harness on the worker host, written for this pass (not the author's scripts): `~/d2d-lab/pv3/` (`pv3_setup.sh`, `pv3_build.sh`, `pv3_phase1.sh`, `pv3_phase2.sh`, `pv3_run.sh`, `pv3_probe_build.sh`, probe `pv3_usewarp.cpp`); clean detached worktrees `wt-pv3-a` (as pushed: `8d7b89f1`, `feba9539`) and `wt-pv3-r` (rebased: `c9b4c06e`, `f0e8b396`), builds `build-pv3-{a,r}`. The oracle is `vb2_tele.py` (copied unchanged from `tools/emu/`): FUN_0064dea0 run natively, the room lookup and the grid test hooked.

## Verdict and required edits

| # | Severity | Finding | Edit |
|---|---|---|---|
| 1 | HIGH (blocks posting) | No test. Upstream AGENTS.md section 4.4: "Fix or port: the change and a test in `tests/`, citing the game.exe address." | Add `tests/game/test_warp_arrival.cpp` (`// test-links: game`): (a) synthetic: a `Level` whose unit subtile is blocked so that `nearest_free` and `warp_spot` disagree (any `differ` case of `vb2_tele.py gen 7`), assert `World::warp_arrival` gives game.exe's subtile; (b) data part (SKIP without MPQs, as `test_warp_click`): `World::use_warp` on one seed through a warp whose unit is blocked (seed 5 has 28), assert the hero's subtile equals the game.exe answer recorded in the test. |
| 2 | HIGH (blocks posting) | No oracle in the PR. Section 5: "The oracle goes in the same pull request as `tools/emu/<name>.py`". `sk_tele_spot.py` (upstream) proves `warp_spot` for Teleport through FUN_00554ea0, but not the warp's own first call (FUN_005550b0 0x555144..0x555157: size 2, mask 0x1c09, flag 1) nor d2d's warp path. | Oracle-first commit as #444 / #445: a `warp` mode in `sk_tele_spot.py` (or `tools/emu/warp_arrival.py`) that runs FUN_0064e7b0 with FUN_005550b0's arguments then FUN_00554ea0, natively including FUN_0064d9b0, on synthetic rooms; the test's oracle mode prints d2d's `warp_arrival` for the same cases; README command in the PR body; before 10,265 / after 0 style numbers. |
| 3 | MEDIUM | No research-doc change. `docs/research/re/drlg.md` line 1623 says "warp arrival traced, not diffed", and "Taking a warp" cites FUN_00554ea0 for the spot; the spot is searched twice (below). | In the same PR: drlg.md "Taking a warp": the first search in FUN_005550b0 (flag 1: no spot returns the unit's own subtile), the second in FUN_00554ea0 (flag 0: no spot fails, so the warp is not taken); the ring rule and range; row 1623 to "diffed (tools/emu/...)". |
| 4 | MEDIUM | The fallback contradicts the claim. When `warp_spot` finds nothing in rings 1..49, `warp_arrival` falls back to `nearest_free`, so d2d lands the hero; game.exe does not take the warp (V read, below). Not reached on real levels here (game.exe found a spot on all 452 real arrivals of each tree), reached in 426 of 20,000 synthetic cases. | Either return `std::optional` from `warp_arrival` and `return` from `use_warp` without arriving (as the shut sewer stairs do, with the 0x55515e / 0x5551c3 citation), or keep the fallback with a `// ponytail:` note saying game.exe refuses the warp there. The first is the faithful one and is two lines. |
| 5 | MEDIUM | The comment and commit cite the wrong caller. The spot the player lands on comes from FUN_005550b0's own call of FUN_0064e7b0 (0x555157); FUN_00554ea0's call (0x554f43) re-checks it. The comment says "FUN_00554ea0's spot". Same rule, so no code change. | Comment: "FUN_005550b0 -> FUN_0064e7b0 -> FUN_0064dea0 (0x555144..0x555157; FUN_00554ea0 searches again from that spot and finds it free)". |
| 6 | LOW | Commit message format. Subject 92 characters (upstream median 56, p95 78); prefix `game:` is not upstream's style; body 6 lines (section 6: at most three); author `Claude`; no `Agent:` trailer. | New branch `fid/warp-arrival-r2` off current upstream main (oracle commit, then claim commit), authored by the user's GitHub identity, e.g. subject `A warp lands on the free subtile nearest its unit, as FUN_005550b0`, three body lines, blank line, `Agent: Claude Code`. Never force-push r1. |
| 7 | LOW (disclose) | One residual on other acts: `World::arrive` re-places the hero with `nearest_free` after `init_new_objects` / `reopen_doors` (travel.cpp, "off what the room just made"). Where an object made at arrival covers the spot, d2d's ring order applies again: 1 of 139 arrivals (Act 2, level 51 to 52, seed 5). Not in Act 1 (0 of 312). | Name it in the PR body's limits ("an object made on arrival: arrive's second nearest_free, 1 of 139"), or route that re-place through `warp_spot` too (a second change; not needed for this PR). |
| 8 | LOW (disclose) | `warp_spot` tests the grid without the units' bit (mask 0x0c09; game.exe passes 0x1c09, which includes units standing there). A monster on the exit subtile moves game.exe's hero, not d2d's. Pre-existing (Teleport has the same), d2d keeps units in the Crowd. | One line in the limits. |

With 1 to 6 done and the re-check below re-run on the new branch, this is a PASS on the evidence here. Nothing here touches the bot; the bounce (fid-warp-arrival.md Q3) is unaffected, as the author said.

## 1. Differential (V run, clean worktrees)

`pv3_usewarp` is the FULL path: a `World` built per seed, the player put on each warp's unit of every Act 1 level, `take_warp` set, `World::use_warp()` called (which calls `warp_arrival` and `World::arrive` with its rooms, objects, doors and second `nearest_free`); the hero's final subtile is compared with game.exe's FUN_0064dea0 on a 200 x 200 window of the destination's collision, taken before and after the arrival. The author's helper probe (`d12_arrive_probe`: A `nearest_free`, B `warp_spot`, C `warp_arrival`) was rebuilt against the same trees as a cross-check.

Act 1 (levels 1..39, 52 arrivals a seed, 28 of them with the unit subtile blocked), arrivals whose final subtile differs from game.exe's:

| seed | parent as pushed | claim as pushed | parent rebased | claim rebased |
|---|---|---|---|---|
| 5 | 26 | 0 | 26 | 0 |
| 82 | 26 | 0 | 26 | 0 |
| 7 | 25 | 0 | 25 | 0 |
| 11 (new) | 25 | 0 | 25 | 0 |
| 23 (new) | 26 | 0 | 26 | 0 |
| 4242 (new) | 25 | 0 | 25 | 0 |

- The same numbers against the pre- and post-arrival windows (game.exe's answer is the same on both in Act 1, 0 of 312 differ), and the helper probe agrees: A = the parent column, B = 0, C = 0 on the claim. The author's 26 / 26 / 25 reproduce exactly.
- Acts 2 to 5 (levels 40..136, seed 5, 140 arrivals, 113 moved off the unit): parent 92 differ, claim 2 (post window) / 3 (pre window), as pushed and rebased. One is the Kurast Sewers stairs (92 to 93), which d2d keeps shut until the lever (the quest gate, not a spot difference). The other is finding 7 (level 51 to 52: the pre-arrival spot (2, 0), an object made on arrival blocks it, game.exe on the post window (2, -1), d2d's re-place (3, -1)). Level 52 to 53 matches by luck of the ring order. So the change fixes far more than Act 1: the PR can say "every act".
- Synthetic (`vb2_tele.py gen`, 20,000 cases a seed): `nearest_free` differs 10,265 (seed 7, the author's number) and 10,128 (seed 31, new); `warp_spot` 0 and 0. Of these, 426 / 395 are cases game.exe refuses (finding 4).

## 2. game.exe chain (V read here)

- FUN_005550b0: the tile unit by FUN_006195a0; its room FUN_00620bb0, its x / y FUN_0045adf0 / FUN_0045ae20; then `FUN_0064e7b0(room, &xy, FUN_00620510(player), 0x1c09, 1)` (0x555144..0x555157); 0 returned: the function ends (0x555160, no warp). Then the quest gate FUN_00545b80 for levels 0x49, 0x80, 0x84, 0x76, 0x64; then `FUN_00554ea0(room, x, y, 0, 0)` (0x5551bc); 0 returned: no warp (0x5551c3); then the ExitWalk walk FUN_005809d0 and the 0x0d packet.
- FUN_00554ea0 with its fourth argument 0: `FUN_0064e7b0(room, &xy, size, 0x1c09, 0)` again (0x554f30..0x554f43), then FUN_00650be0 places the unit.
- FUN_0064e7b0 (0x64e7b0..0x64e7d4): `FUN_0064dea0(a1, a2, 0, a3, a4, a4, a5, 0x32, 1)`.
- FUN_0064dea0: the start subtile in a room and free (FUN_0064d9b0 with size and mask) is returned as is. Else rings r = 1, 2, ... while r + 1 < 0x32 (0x64e1e5..0x64e1f8): rings 1 to 49. In a ring, first the columns x - r and x + r for rows y - r .. y + r (corners included), row by row, left first (0x64dfe0..0x64e0ca); then the rows y - r and y + r for columns x - r + 1 .. x + r - 1, column by column, top first (0x64e0d0..0x64e1bd). A free subtile in a room is kept only when |dx| + |dy| is strictly less than the best so far (`cmp eax, ecx; jge`, 0x64e088 / 0x64e181): ties go to the first in that order. The first ring with a free subtile ends the search. None in 49 rings: with the flag (+0x20) set, the start's room is returned with xy unchanged (0x64e22d..0x64e24a); without, 0.
- So for the warp: no spot in 49 rings, the first call returns the blocked unit subtile, FUN_00554ea0's second call returns 0, and the player is not moved: the warp is refused (finding 4).
- d2d `Level::warp_spot` scans the corners last, game.exe inside the columns; a corner's |dx| + |dy| = 2r is larger than any other subtile of the ring, so it can only win when nothing else is free, and the four corners keep game.exe's order. Same result (0 of 40,000 synthetic, 0 of 312 real Act 1 arrivals here).
- Blocked: FUN_00620510 gives 2 for a player (jump table 0x62057c); FUN_0064d9b0 size 2 goes to FUN_0064d100, whose in-room case (0x64d1b6) ORs the words at the subtile, left, right, up and down and ANDs the mask: the plus footprint `unit_blocked` uses. Mask 0x1c09 = walls 0x0001, 0x0008, 0x0400, 0x0800 and the units' 0x1000 (finding 8). The oracle hooks this test with the plus (S in `vb2_tele.py`); the read here backs it for subtiles whose neighbours are in the same room (the room-edge cases 1..12 of FUN_0064d100's table were not read).

## 3. Regressions (V run)
- The diff changes one caller: `use_warp`'s call of `arrive`. No other caller of `nearest_free` or `arrive` changes (the 50-odd `nearest_free` callers in quests, setup, objects, ai, loot, travel are untouched; waypoints, portals and quest arrivals keep `nearest_free`). `arrive`'s first `nearest_free` gets a free spot and returns it unchanged; merc, pets, footstep and `warp_spot` follow the new spot (intended).
- `ctest -j8` (D2_MPQ_DIR and the 1.14d installer set) on all four builds: the same 4 failures on parent and claim, as pushed (248 tests) and rebased (273): `test_mpq`, `smoke_d2d` ("no voice", d2speech), `quests_act1` / `quests_act4` (time out at the quake camera); the log warns `no d2video.mpq` / `no d2xvideo.mpq`. Same set as the #444 / #445 verification. No test exercises a warp arrival (finding 1).

## 4. Builds and hygiene (V run)
- Full `ninja` (all targets, `D2D_WERROR=OFF`) at each commit alone: `8d7b89f1`, `feba9539`, `c9b4c06e`, `f0e8b396`: all rc 0.
- `pr_guard.py HEAD~1` on the rebased claim: 0 FAIL, 0 WARN; `--sorted`: 0 FAIL.
- PRESUBMIT_LINE
- The diff (2 files, 12 added lines) holds no host, path, fork verb or lab word (V read). One topic, minimal.

## 5. What was not done
- game.exe's whole FUN_005550b0 on a real level (room lists and unit lists hooked away); the spot rule was run natively on d2d's collision of the real levels, whose equality with game.exe's grid is S (fid-warp-arrival.md, 100 of 131 levels).
- FUN_0064d100's room-edge cases; units in the grid (finding 8).
- No live run, no sandbox, nothing posted, no push to GitHub.

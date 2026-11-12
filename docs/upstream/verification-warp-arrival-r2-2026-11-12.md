# Second independent verification of warp arrival r2 (2026-11-12)

> status: current, 2026-11-12 (TASK-376, agent prverify4). Verdict **PASS** (two LOW wording edits to the draft, not blocking). Nothing posted, nothing pushed. Consumer: TASK-45 (upstream PR posting); the user's push go is still needed.

Verifier: d2-reviewer (Opus), singly, second pass. Candidate: `fid/warp-arrival-r2` on the private fork remote, head `24e8ac4f`, two commits on upstream main `c9b4c06e` (oracle `688e53ed`, claim `24e8ac4f`). Draft: [drafts/warp-arrival.md](drafts/warp-arrival.md); queue row 1a in [pr-queue.md](pr-queue.md). First pass (NEEDS-WORK on r1): [verification-warp-arrival-2026-11-12.md](verification-warp-arrival-2026-11-12.md). Tags: V read, V run (on the worker host, here), S, U.

Harness (written for this pass, scratch, not for upstream): `~/d2d-lab/pv4/` on the worker host: `pv4_build.sh`, `builds.sh`, `pv4_usewarp.cpp` (copy of the first pass's full-path probe; it also runs and reports warps with no way back instead of skipping them), `pv4_run.sh`, `to_vb2.py`, `failvariants.sh`, `ctests.sh`; clean detached worktrees `wt-pv4-{p,o,c}` (parent, oracle, claim), `wt-pv4-e` (Python only), `wt-pv4-f` (patched variants); builds `build-pv4-{p,o,c,f}` (Release, asserts kept on by `tests/CMakeLists.txt`). Independent oracle: the first pass's `vb2_tele.py` (FUN_0064dea0 called directly, grid test hooked with the plus), not the PR's.

## Verdict

PASS. Every required edit of the first pass is applied and true (table below); all numbers reproduce on new seeds; the PR is one topic with a minimal diff; upstream has not moved and nothing open conflicts. Two LOW wording edits to the draft (below) can go in with the push; neither touches the branch.

| # | First-pass edit | r2 state | Evidence |
|---|---|---|---|
| 1 | test in `tests/` | DONE. `tests/game/test_warp_arrival.cpp` (`// test-links: game`, addresses cited): 10 hand cases, whole `use_warp` on seed 5 through 10->5, 15->11, 17->18, refusal with the destination fully blocked. Passes on the claim; **fails on the parent behaviour** both ways (next section) | V run |
| 2 | oracle in the PR, first commit | DONE. `688e53ed` adds `sk_tele_spot.py warp` / `gen --warp` (calls game.exe's FUN_0064e7b0 then FUN_00554ea0 natively, FUN_0064dea0 and FUN_0064d9b0 included) and the README row; the test file's oracle mode prints the parent's `nearest_free` answer; builds alone | V read, V run |
| 3 | drlg.md | DONE. "Taking a warp" now has the two searches (flag 1 at 0x555144..0x555157, flag 0 via FUN_00554ea0 at 0x5551bc, refusal 0x5551c3), the ring rule and range; the table row says "diffed (`tools/emu/sk_tele_spot.py warp`)". Matches the first pass's own reading of FUN_005550b0 / FUN_0064dea0 | V read |
| 4 | refuse, no fallback | DONE. `warp_arrival` returns `std::optional`; on nullopt `use_warp` logs `no free spot within 49 subtiles ... the warp is not taken` and returns. `take_warp` is cleared earlier (`actor->take_warp = -1` before the sewer gate), nothing else has been mutated, `arrive` is not called, so hero, level, merc and pets stay. The test asserts level, position and `take_warp < 0` | V read, V run |
| 5 | cite FUN_005550b0 | DONE in `world.hpp` comment, commit subject, drlg.md | V read |
| 6 | commit format | DONE. Both commits: author and committer the user's GitHub noreply identity, subjects 46 and 66 characters, 3 and 2 body lines, blank line, `Agent: Claude Code`; no Co-Authored-By or Claude/Cursor trailers | V run (`git log --format`) |
| 7, 8 | disclose residual, units' bit | DONE in the draft's limits | V read |

## Numbers on new seeds (V run, seeds not used by the author or the first pass)

| Run | Parent (`c9b4c06e`) differ | Claim (`24e8ac4f`) differ |
|---|---|---|
| Synthetic, PR oracle, `gen --warp 20000 613` (19,740 land, 260 refused in game.exe) | 11,352 (11,092 other subtile, 260 d2d lands where game.exe refuses), via the oracle commit's `--warp-arrival` | **0** (`20000 same`) |
| Act 1, full `use_warp`, seed 101 (52 arrivals, 28 moved) | 25 | **0** |
| Act 1, seed 2027 | 26 | **0** |
| Act 1, seed 9001 | 25 | **0** |
| Acts 2-5 (levels 40..136), seed 77 (139 arrivals, 114 moved) | 94 | **2** (post-arrival window; 3 on the pre-arrival window) |

- Oracle cross-check: the same 20,000 seed-613 cases converted to the first pass's independent `vb2_tele.py` (cells with a bit of 0x0c09 blocked) give a byte-identical answer file to `sk_tele_spot.py warp`. The PR's oracle (two calls, native grid test) and an oracle written separately (one call, hooked grid test) agree on every case, including the 260 refusals.
- The two Acts 2-5 residuals on seed 77 are exactly the two the draft names: level 92 to 93 (Kurast Sewers stairs, d2d's quest gate keeps them shut, not reached) and level 51 to 52 (claim lands (3, -1) from the unit, game.exe on the post-arrival window (2, -1): an object made on arrival, `arrive`'s second `nearest_free`). The third pre-window difference is 53 to 54, where game.exe's own pre and post answers differ (an object made on arrival), and the claim matches the post window. Same pattern as seed 5: the limits are not overclaimed and hold on another seed.
- No arrival on seeds 101/2027/9001/77 has no way back (0 skipped), so the `back == end` path (arrive at the level centre, now through `warp_spot` and able to refuse) was not reached; it has no game.exe counterpart. U, harmless.
- The test fails on the parent behaviour (V run, `failvariants.sh`): v1, claim tree with `warp_arrival` returning `nearest_free` (the parent's rule): aborts at the third hand case (`test_warp_arrival.cpp:102`); v2, claim tree with `use_warp` arriving at the unit as before: aborts in the whole-`use_warp` part (`:154`, `reached && dx == want_x && dy == want_y`). The unmodified claim passes (`OK: a warp lands on game.exe's spot and is refused when there is none`), including the refusal case with the log line. The test cannot run unchanged on the parent tree itself (it calls `World::warp_arrival`, new in the claim), which is the normal shape for a new function.

## Builds and hygiene (V run)
- Full `ninja` (all targets, `D2D_WERROR=OFF`) at each commit alone: `c9b4c06e`, `688e53ed`, `24e8ac4f`: rc 0.
- `pr_guard.py c9b4c06e HEAD`: 0 FAIL, 0 WARN; on the oracle commit alone 0 / 0; `--sorted c9b4c06e`: 0 FAIL. (Against the lab clone's stale `origin/main` `--sorted` reports a `units.md` row not in this diff; ignore.)
- `presubmit.py --quick --base c9b4c06e`: 0 FAIL, 1 WARN (`build` skipped by `--quick`); GCC `-Wshadow -Wconversion -Werror` and clang-tidy on 174 TUs ok; ruff ok.
CTEST_PLACEHOLDER
- Lab content: the diff has no `lockstep`, `devctl`, `framefeed`, lab verb, cheat, host, path or name (grep over `git diff c9b4c06e 24e8ac4f`, 0 hits). The test's default data dir `~/Workspace/private/diablo2` is upstream's own convention (`test_akara_decipher.cpp` and others).
- One topic, minimal: code 11 added lines in `world.hpp`, 6 added and 1 removed in `travel.cpp`; one caller changes (`use_warp`); waypoints, portals, quest arrivals and `arrive`'s own `nearest_free` untouched.

## Upstream state (V run, read-only `gh`, 2026-11-12)
- Upstream main is still `c9b4c06e`: the branch is current, no rebase needed.
- Our open PRs upstream: none. #445 and #444 are merged (#445 is `c9b4c06e` itself), so the one-open-PR-per-file and CodeRabbit limits leave room.
- Open PRs touching the same files: #483, #484, #433 (`world.hpp`), #476 (`tools/emu/README.md`). `git merge-tree` of r2 with each of them and with #486 ("Snap an empty walk onto the subtile centre", the nearest topic): all clean. No open PR or issue covers warp arrival (search "warp": only merged #269, #234 and issue #83, unrelated).
- Upstream AGENTS.md rules checked: oracle in the same PR as `tools/emu/<name>.py` (here an extension of `sk_tele_spot.py`, acceptable as #444 / #445 did); a code change's commit carries its test and cites the address (claim commit); short subject, at most three body lines; author is the person; `Agent:` disclosure; leave `docs/PLAN.md` alone (untouched); presubmit before pushing.

## Edits (LOW, draft only; the branch needs none)
1. Draft "What changes": "5 in `world/travel.cpp`" is 6 added, 1 removed. Say "6 added lines (1 removed)".
2. Draft numbers: optional, add this pass's new seeds (synthetic 613: 11,352 before, 0 after; Act 1 seeds 101, 2027, 9001: 25 / 26 / 25 before, 0 after; Acts 2-5 seed 77: 94 of 139 before, 2 after, the same two residuals). Not needed for posting.
3. Opinion (S): the commit bodies are three and two lines but each about 150 characters; upstream's rule is lines, not width, so it passes; a maintainer may prefer wrapping. Leave as is.
4. Opinion (S): the refusal test writes through `data->level(5)`'s `mutable` `walk` and leaves Dark Wood fully blocked for the rest of the process. It is the last step of `main`, so harmless; CodeRabbit may comment. If it does, copy the level or restore the vector.

## Not done
- No live run, no sandbox, nothing posted, no push to GitHub or the fork. Multi-room ring behaviour and FUN_0064d100's room-edge cases stay U, as the draft says.

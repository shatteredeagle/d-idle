# d2d lab6 (private fork rebase, agent FORKREBASE, 2026-10-30)

> status: current record of the lab6 build. Overview in docs/d2d-fork.md section "lab6". Patch content stays on the worker host and the forge `<owner>/d2d-fork`, never here (GPL).

Tags: V = verified on the worker host this session (how stated), S = read in source, U = not verified.

## Why

lab5 sits on upstream 3306e60. Upstream moved 22 commits, among them the `kill_exp` fix 01337ba3 (docs/research/xp-award-trace.md: the old rule was wrong from monster-level diff 6 up) and our own PRs #127 (vendor prices) and #128 (cast ticks). lab6 = current upstream main (32768ddb) plus the lab patches that are still ours.

## Patch inventory (lab5, commit by commit) and what lab6 did with it

| lab5 commit | What | lab6 |
|---|---|---|
| 7196eb2 P2 | virtual clock, `--lockstep`, `--time-scale`, `step`/`clock`/`lockstep`/`timescale`, deferred devctl reply | kept (1 conflict: CMake source list, tile_blit.cpp is new upstream) |
| d579f16 P1 | `dump` verb (player, stats, skills, items, quests, ui, merc, time) | kept |
| 23e8d2c | `dump vendor` | kept |
| c17e42f P3 | `debug layout`, `debug nearby` | kept |
| 3c48fdf | prices of tomes and quivers by quantity | DROPPED: upstream PR #127 |
| 3cbf274 | cast timing | DROPPED: upstream PR #128 |
| b99a6dc | `cmd belt put`, `cmd tobelt` | kept |
| 173cbc9 | `dump ui` rects | kept |
| 8b649b5 | `debug wp all`, `debug wp level <id>` | kept |
| 38a7d59, 7ed8f1c, 4a90173, 217dd20 | lockstep speed: 1 ms idle floor, 10 fps draw cadence, no render except `step n render` or `screenshot`, poll()-driven devctl | kept |
| new | lockstep: make level linking deterministic (DETLOOP request): under lockstep `poll_levels` waits for pending builder jobs, so a wanted level installs on the next tick whatever the wall time; outside lockstep unchanged (a global flag set by `--lockstep` and `lockstep on|off`) | added |

The price and cast patches were dropped, not re-checked by diff: upstream merged the same PRs (merge commits 32768ddb, af825eda in lab6's history), S. The upstream test_combat and test_game cover the cast/price rows (V below).

Cherry-pick order on `upstream/main`: 11 picks, one conflict (CMake list), then the new commit. Fork repo `~/d2d-lab/fork/src` got a remote `upstream` (https://github.com/psi29a/d2decomped.git, fetch only); worktree `~/d2d-lab/fork/wt-lab6`, build dir `build-lab6` (Ninja, Release, `-DD2D_WERROR=OFF`, deps reused from `src/build/_deps` for CLI11, sentry_native, StormLib), binary `bin/d2d.lab6` (mode 555, sha256 in `bin/d2d.lab6.txt`).

## Verification (V unless noted)

- Upstream tests: `test_combat` OK, `test_game` OK (one data-dependent block SKIPs: no d2data.mpq at the default path), built from lab6.
- Scripted scenario `det6.py` (copy of det.py: own slot dir and socket, seed 3, Blood Moor, 8 x 500 steps, md5 of the `monsters` lines):
  - passive zombies: 3/3 identical, same hash as lab5 (`384286bc974a`).
  - FIGHT=1 (clvl 30, warped next to quill rats): 3/3 identical (`e66c585ad1fd`); lab5 gave `9b0e4358e671` (3/3). They differ, as expected after 22 upstream commits touching combat and XP; the cause was not bisected (U).
- Verbs: `verify-lab3.py` run with lab5 and lab6 gives the same 66 output lines except the start-up frame counter (clock, rects, belt put/tobelt, `use`, `debug wp`, `dump quests`, `dump ui`). `debug layout`/`nearby`, `dump` and `step` also ran in the bot pair runs below. `screenshot` not re-run (U).
- Bot smoke and determinism: DETLOOP's `pair.sh` with lab6 (`pair6.sh p6 d2bot.det3 23 12000 8m 5`, slot wrapper): five full runs of the campaign bot to tick 13117, no crash. Compared with `d2ddet compare -ignore state`: A..E are record-identical (including all checkpoints) up to tick 2250..2502 and diverge at tick 2394, 2556, 2619 (BOT-SIDE per d2ddet: the world replies are identical, the bot sends `debug grid 5` in one run only). So the first divergence did NOT move past 3068/3230 for this seed (the seeds differ, so not directly comparable); what changed is that no world-side difference appears before it. The `state` reply differs between runs at tick 8 to 143 (startup, source 6 in docs/d2d-determinism.md) and is masked by `-ignore state`. The remaining refetch decision is in the bot (`d2dworld`), DETLOOP's domain.
- Speed (lockstep, Blood Moor, 20000 steps x3, alternating lab5/lab6/lab5/lab6 in one slot under load 12): lab5 17.3k to 18.3k steps/s, lab6 18.4k to 19.4k steps/s (about 6 % faster, within the 10 % limit).

## Migration

- `D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab6` or `D2D_LAB=lab6` with `~/d2d-lab/fork/start.sh` (the default stays lab2; lab5 stays for comparison). Use lab6 for runs past clvl 7 (correct XP). Do not mix binaries within one run: a different build is a different world (docs/d2d-determinism.md section 5).
- Hash changes vs lab5 are legitimate wherever XP, cast timing or combat changed; passive scenarios are byte-identical.
- Slots: runs of the verification used `~/d2d-lab/d2dslot.sh run -w 3000 -- ...`. Never `pkill -f d2d.labN`: other agents run the same binary name; kill your own pid.

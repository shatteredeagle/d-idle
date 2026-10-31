# Soak judging and game recycle (agent SOAKFIX)

> status: Game.exe judge and recycle; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md). Still the judge of the frozen path; the d2d replicated soak and gate v2 are docs/d2d-soak.md and openspec/specs/soak-gates.

Tags: V verified offline (tests named below), U unverified live.

## How d2soak judges slopes (cmd/d2soak/analyze.go)
- **Warm-up** (`--warmup`, default 10 min, 0 = off): the first WarmUp of every slope series is dropped, and again after each
  process restart (a recycled Game.exe repeats its climb: pid/start change starts a new segment `name#2` with its own warm-up).
  Slopes need at least `--min-slope-span` (30 min) and 10 samples of post-warm-up data; shorter runs get the `short-run` warning,
  never a fail. V `TestWarmUpDropsEarlyGrowthOfOtherSeries`, `TestRestartSegmentGetsOwnWarmUp`.
- **fd plateau**: fds step up on the first visit of each new area (about +70, all `anon_inode:ntsync`) and then stay flat
  (docs/leakhunt.md, VERIFIED LIVE). For fd series the fit ignores steps: an increase of at least `--area-step-fds` (8) in an
  interval next to a sample that saw an `area_change` event is removed; small increases there stay, so a leak across area changes
  is still seen. V `TestPlateauAfterWarmUpPasses` (the LEAKHUNT shape: 54 -> 126 in 2.5 min, +14, +24 at later area changes:
  slope about 0, passes), `TestGenuineLeakStillFails` (+50/h on top still fails with slope about 50),
  `TestPlateauWithoutHandlingWouldFail` (control).
- The report prints `warm-up` and one `fd plateau` line per fd series: value at end of warm-up -> end, steps ignored (count and
  total fds), residual slope. JSON: `warmUp`, `fdPlateau`.
- Needs the bot to emit `area_change` events (it does: goals.EvAreaChange / event.KindAreaChange, counted per sample by kind).

## Profile thresholds (all absolute, per hour, after warm-up; U until a 24 h run calibrates them)
| profile | rss MB/h | fds/h | threads/h | goroutines/h | why |
|---|---|---|---|---|---|
| default | 50 | 5 | 10 | 10 | starting points |
| m1 (2 h) | 60 | 15 | 10 | 10 | short run: wineserver/d2input still gain about +30/h in the first hours (LEAKHUNT), little data |
| m3 (24 h) | 40 | 8 | 5 | 5 | fd slope is mostly plateau now; real leaks of 10/h are 240 fds in a day |
| v1 (72 h) | 30 | 5 | 3 | 3 | long enough that anything above is a leak |
Roadmap's "RSS < 5 %/h" is relative: set `--max-rss-mb-per-hour` to 5 % of the settled game RSS from the first report. Explicit
flags always beat the profile.

## Recycler hooks (internal/launch/recycle.go, V internal/launch/recycle_test.go)
- `RecyclePolicy.Due(RecycleStats{Cycles, FDs, Last, Now}) (bool, reason)`: pure, no clock, no I/O. FDs < 0 = unknown.
  `Recycler.Stats()` samples, `Recycler.DueFor(stats)` decides, `Recycler.Due()` = both (signature kept: the old two-value form).
- `launch.NewRecycler()` / `(*Recycler).FromEnv()`: policy from `D2BOT_RECYCLE_CYCLES`, `D2BOT_RECYCLE_FDS`,
  `D2BOT_RECYCLE_MIN_INTERVAL` (defaults 500 / 4000 / 30 m, 0 disables; the policy is a safety net: 4000 fds is far above the plateau).
- `Recycler.Status() RecycleStatus` (JSON): enabled, recycles, cyclesSinceRecycle, maxCycles, fds, maxFds, lastRecycle,
  nextEligible, due, dueReason.

### Between-goals call sequence
1. A goal ends (done, failed, abandoned) and the bot is in town, no fight, no panel open, no pending trip. Never mid-goal.
2. `rc.NoteCycle()` for each leave/re-enter the bot did itself (the save-cycle knob G-2, chicken exit).
3. `ok, err := rc.Maybe(ctx)`. `ok == false`: continue with the next goal. `ok == true, err == nil`: the game is fresh and logged in;
   re-attach memory/input readers (the Login hook does it), re-read character state, continue.
4. `err != nil`: the recycle half-ran (the game may be down). Do not call again at once (MinInterval guards it); hand to the failsafe
   ladder rung 4 (`Deps.Restart`) and report `recycle_failed` as an event.
5. Wire the hooks as: `FDs` = `launch.CountFDs(gamePID)`; `Leave` = Flow.LeaveGame; `Stop`/`Start` = launch.Life; `Login` = attach + Flow.Start(char).

## Requests for the owners (not done here: not my directories)
- **cmd/d2bot (INTEGRATE/bot owners)**: build one `launch.NewRecycler()` per instance in `--managed`, fill the hooks above, and call
  `Maybe(ctx)` from the goal runner's "goal finished" path before the next goal starts. Emit an event `recycle` (reason in the
  payload) so the soak report can label the restart; the game pid change already shows as `processRestarts` for `game`, and the
  sentinel must not count it as a manual restart.
- **internal/bot**: call `NoteCycle()` where it leaves and re-enters (save cycle, chicken re-entry); expose a `Recycle` hook for rung 4.
- **internal/fleet**: one Recycler per instance; stagger so only one instance recycles at a time (the worker host load). Include
  `Recycler.Status()` in the per-instance status JSON (`GET .../status` -> `recycle`) and show cycles, fds, next eligible on the
  dashboard card.
- **bot `/metrics`**: expose `game_open_fds` (= `CountFDs`) as a `GaugeFunc` so the soak can chart it without /proc access.

## Live checklist (U, needs the game: not run here)
1. `tools/soak/soak.sh start i4 --for 2h --profile m1`, then `soak.sh report`: expect the `fd plateau` line with a small residual slope.
2. After the run: `d2soak report --in samples.jsonl --warmup 0 --area-step-fds 0 --json` shows what the old judge would have said.
3. Once wired: set `D2BOT_RECYCLE_CYCLES=3` for a short drill, expect one `game` restart in the report and no `leak`.

## Recycler wired (INTEGRATE2, 2026-10-28; offline V, live U)
- `cmd/d2bot/recycle.go`: managed mode builds one `launch.Recycler` per instance (`newRecycler`): Leave = `Flow.LeaveGame` (it also remembers the difficulty), Stop = `Game.Shutdown`,
  Start = helper `Ensure` + `Game.Launch` + `Attach` (the bot reads through a `swapSource`, so the new Game.exe is read without rebuilding any reader), Login = `Cycler.EnterGame`
  (which resets the map tracker). `cycler.onCycle` calls `NoteCycle` for every leave/re-entry the bot itself makes. Attach mode has no recycler.
- `bot.Deps.RecycleDue` (cheap check) and `bot.Deps.Recycle`: the runner asks when a NEW goal starts (before `resetForGoal`, so before any goal action) and, with no goal, once a minute; only in
  town, alive, nothing hostile near, no panel open. After a restart it clears the "game gone" latch (the old process exiting is the plan), logs `game recycled` and publishes an event of kind `lifecycle`
  with state `recycle` (or `recycle_failed`, reason in the payload); `Deps.Restart` (failsafe rung 4) is `Recycle(ctx, "failsafe restart")`. A campaign goal is one long goal: it is checked only at its start.
- `/metrics`: `game_open_fds` (Game.exe descriptors, -1 when not running) and `game_recycles`. `GET .../bot/state` carries `recycle` (`RecycleStatus`: cycles, fds, next eligible, due).
- Not done: fleet-level staggering (one recycle at a time), the dashboard card. Live drill (`D2BOT_RECYCLE_CYCLES=3`) not run.

## Per-instance process match
`match:SUBSTR` takes the lowest pid, wrong when several instances run (the 2026-10-29 i4 soak's
`proc:wineserver:fds` may have been another instance's). Use `find:comm=NAME;cmd=SUBSTR;env=KEY=VAL`
(exact comm, cmdline substring, exact environ entry; all given terms must hold). `tools/soak/soak.sh start iN`
builds these from the instance number; `soak.sh procs iN` (or `d2soak procs --proc ...`) is the dry run that
prints the resolved pids. V (read-only /proc on the worker host, 2026-10-29): Game.exe and wineserver have
D2BOT_INSTANCE=iN in environ; d2input.exe has `-addr 127.0.0.1:47012+N`; Xvfb is `:10N`; d2bot has
`--instance iN` (the wrapper bash/timeout lines have other comms). An `--attach` bot has no --instance: MISSING.
Limit: i0 falls back to `match:` for game/wineserver/helper (U: i0's environ not checked).

## Status, alerts and report fixes (agent SOAKTOOLS, 2026-10-29; offline V, live U)
- **Level 0 -> 0**: V (replayed the real i4 samples of 2026-10-29: 40-sample run had level 0 in the first and last sample, `/status` failing
  at the edges, the rest 4). The report used the first and last sample literally; it now uses the first and last sample with a level
  (`levelRange`), and prints `?` when none is known. Test `TestLevelSkipsSamplesWithoutLevel`.
- **Empty run said FAIL**: with fewer than 2 samples the report verdict is `NO-DATA` (`verdict` in JSON, `noData`, finding level warn), not
  FAIL; exit code 3 (0 pass, 1 fail, 2 usage). Test `TestNoSamplesIsNoDataNotFail`. `soak.sh report/alerts/status` say `no soak runs yet` without a run dir.
- **`start` printing inactive**: S (cause not reproduced, no live run): `d2soak run` exits at once when the bot is not reachable yet
  (`Resolve` error), and the script read the unit state once after 2 s. Now `run` retries for `--wait-target` (default 2 m), and
  `soak.sh start` polls up to 20 s for `active`, else prints the journal tail and exits 1. The binary is copied as `d2soak.new` and renamed (no
  "text file busy" over a running soak), and skipped when the sha256 matches.
- **`soak.sh status`**: unit list plus `d2soak status -in RUNDIR`: `soak 1h0m0s: lvl 4, 5042 xp/h, deaths 0, failsafe rungs 1, restarts 1.0/h,
  helper 59 fds/8 thr, areas [A > B > C], 10m since progress` (adds `LAST SAMPLE nm AGO` when the sampler stalls). Area names come from `/status`
  `runner.character.area`, now stored in the sample (`area`, `charXp`); samples of older runs have no area and show `areas [none]`.
- **Alerts** (`cmd/d2soak/status.go`, `d2soak alerts -in RUNDIR [-write]`, `soak.sh alerts`): `stall-xp` (no XP change > `--alert-stall` 10 m, idle/paused exempt),
  `restarts` (> `--alert-restarts-per-hour` 8 process pid/start changes in the trailing hour), `fd-slope` (newest segment of each fd series over
  `--max-fds-per-hour`, fitted over the trailing `--alert-fd-window` 3 h with warm-up and area-step handling), plus `sampler-stale` (newest sample > 5 m old)
  and `unhealthy`. The running `d2soak run` rewrites `RUN/alerts.json` (atomic rename) and `RUN/status.txt` every 4 samples (~2 min); it also writes
  `RUN/limits.json` so `alerts` judges with the run's profile. Dashboard file: `~/d2lod-live/soak/<run>/alerts.json`,
  `{updated, level ok|warn|crit, alerts:[{level,code,msg}], status}`. The dashboard side (reading it) is not done here.
- Runs started before this change have no `alerts.json`; `d2soak alerts -in RUN` computes it on demand with default rules and limits.
- Live checklist (U): `tools/soak/soak.sh start i4 --for 2h --profile m1` on a free instance, wait 3 min, `soak.sh status`, `cat <run>/alerts.json`.

## Wineserver fd judgement and winedevice reaping (WINEFIX)
- V (docs/leakhunt.md WINELEAK): wineserver fds grow in steps, not continuously. `proc:wineserver:fds` is therefore judged only with >= 2 h of post-warm-up data (`-wine-min-span`; before that the slope shows "insufficient data", never FAIL), fails above 40 fds/h (`-wine-max-fds-per-hour`) or when slope x 24 h exceeds 2000 fds (`-wine-max-growth-24h`). Helper, game and bot fds keep `-max-fds-per-hour` (5). Tests: `cmd/d2soak/wineplateau_test.go`.
- `tools/game.sh reap <id> [--dry-run]` (and automatically after a successful `game.sh stop <id>`, i1..i9 only) runs `tools/winereap.sh` on the worker host; `tools/game.sh reap-all --dry-run` lists candidates of i1..i9 and kills nothing. It kills winedevice.exe of the current user whose environ WINEPREFIX is one of that instance's prefixes (`$(d2_game_root id)/proton/pfx`, plus the new `<data>/prefixes/<id>` and dev-era `~/Games/d2lod-inst/<id>` layouts, `D2_INST_ROOT`, and colon-separated `D2_REAP_PREFIX` for a custom PrefixDir), only when no wineserver (any `wineserver*` comm) of those prefixes runs, re-checked before every kill, all under the instance lock (`game-<id>.lock` new + legacy; real reap bails if busy after 10 s, dry run continues). Fixtures for each layout: `bash tools/winereap_test.sh` (V, fake /proc). U: not run against a real prefix; first live check: `tools/game.sh reap-all --dry-run` and compare with `ps -o pid,ppid,etime,comm`. Not covered: managed d2bot/fleet restarts (soak24, container) do not call winereap; run `game.sh reap <id>` by hand after a soak, or port to internal/launch (review-evening #5, needs a Go owner).

## Soak strip in the dashboard (DASHWIRE)
`server.Config.SoakDir` (`GET /api/v1/soak`, read-only: newest run dir's `alerts.json` + `status.txt`, no new port, 404 when unset) is now set by all three binaries. Default is off everywhere.
- `d2bot --soak-dir DIR` or `D2BOT_SOAK_DIR=DIR`; `d2fleet serve --soak-dir DIR` or `D2BOT_SOAK_DIR=DIR`; `d2dash -soak-dir DIR` or the same env (flag wins over env).
- Start lines for the harness runs (the run dirs live on the worker host, so start the binary that serves the dashboard there; U, not run live, no service was restarted):
  - bot: `D2BOT_SOAK_DIR=$HOME/d2lod-live/soak d2bot --managed --instance i4 ...` (existing flags unchanged).
  - fleet: `d2fleet serve --soak-dir $HOME/d2lod-live/soak ...`.
  - a systemd unit: add `Environment=D2BOT_SOAK_DIR=%h/d2lod-live/soak`, then restart the unit yourself.
  - preview without a game: `d2dash -soak-dir ~/d2lod-live/soak` (simulated bots, real strip).
- Tests (V, offline): `go test -p 1 ./cmd/d2bot -run 'Soak'` (flag/env default off; sim server answers `/api/v1/soak` from a temp dir).
- `tools/game.sh reap <id> [--dry-run]` (and automatically after a successful `game.sh stop <id>`, i1..i9 only) runs `tools/winereap.sh` on the worker host; `tools/game.sh reap-all --dry-run` lists candidates of i1..i9 and kills nothing. It kills winedevice.exe of the current user whose environ WINEPREFIX is one of that instance's prefixes (`$(d2_game_root id)/proton/pfx`, plus the new `<data>/prefixes/<id>` and dev-era `~/Games/d2lod-inst/<id>` layouts, `D2_INST_ROOT`, and colon-separated `D2_REAP_PREFIX` for a custom PrefixDir), only when no wineserver (any `wineserver*` comm) of those prefixes runs, re-checked before every kill, all under the instance lock (`game-<id>.lock` new + legacy; real reap bails if busy after 10 s, dry run continues). Fixtures for each layout: `bash tools/winereap_test.sh` (V, fake /proc). U: not run against a real prefix; first live check: `tools/game.sh reap-all --dry-run` and compare with `ps -o pid,ppid,etime,comm`. Managed d2bot, soak24, fleet and the Recycler are covered by the Go port `internal/launch/reap.go` (`ReapWinedevices`, same rules, exact prefix `<PrefixDir>/pfx` only, SIGTERM like the script): it runs after `Game.Shutdown` succeeded (all managed stops and the Recycler's restart go through it) and after `Life.stopGame` (`d2game stop`, i1..i9). It first waits up to 8 s for the prefix's own wineserver to exit (it lingers after its last client) and refuses if one still runs; re-checks before every kill; skipped for i0 and an unset instance, and on non-Linux. Knob `D2BOT_REAP=0` (`env.ReapEnabled`, default on) turns it off; kills are logged as `reap: killed orphaned winedevice.exe pids ...`. Tests: `go test ./internal/launch -run Reap` (V, fake /proc). U: not run against a live prefix; check after a soak with `ps -C winedevice.exe` and the d2bot log. It does not take the instance lock (d2bot already holds the instance for its lifetime).

## XP source and saved exp (XPLOSS + XPWIRE, 2026-10-29; offline V, live U)
`report.json` `xpSource` says where `xp` / `xpPerHour` come from, and the text report prints it ("XP/h (N XP, from save)"):
- `save`: total experience stored in the character's `.d2s` (`d2soak run|report -save-file PATH`, sample field `saveXp`), as a running maximum. This is the XP that survives a crash, so it is the honest number for a soak.
- `memory`: no save file, but the bot's memory exp (`charXp`), also as a running maximum. Includes XP not saved yet (lost on a game crash).
- `counter`: the gross kill-XP counter of `/stats` (`xpCounter`, `xpCounterPerH` are always filled). It counts XP that was later lost to a crash or restart, and has no exp-table dependency. Used only when neither of the above exists.
When the source is not `counter` the report adds an `xp counter` line (gross) and `xp_lost` (XP the character lost between samples, a warning finding). Alert `xp_at_risk` (`-alert-xp-risk-levels`, default 1, needs `-save-file`) fires when memory exp minus saved exp exceeds one level's worth.
Wiring: `soak.sh start|report` adds `--save-file` itself. Path: `$D2_SOAK_SAVEFILE` if set, else `<game root of the instance>/proton/pfx/drive_c/users/steamuser/Saved Games/Diablo II/<char>.d2s` with `<char>` from `$D2_SOAK_CHAR` (`soak24.sh start` sets it from `--char`), else the newest `.d2s` of that folder. The flag is left out when the file does not exist. `cmd/d2bot` sets `bot.Deps.SavedExp` from the same file (cached by mtime/size), so the bot's `xp at risk` log compares memory exp with the last saved exp. The level-to-exp table (`internal/telemetry`, levels 1-99) is dumped from the real `experience.txt` (V: `data.DB.ExpToLevel` on `/tmp/d2mpq`, the worker host); the old hand-typed table was wrong from level 29.

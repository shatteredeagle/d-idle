# Stall, ladder rung 4 and "the game is gone" (soaks 2026-10-29)

Evidence: i4 soak logs on the worker host `~/d2lod-live/livebot/runs/20261029-021327-d2bot.log` (ended 03:26:48) and
`20261029-032705-d2bot.log` (ended 03:41:29). Tags: V = read in those logs or the code, U = inferred.

## 2. Why the run ended (V)
Rung 4 (`escalate`, internal/bot/loop.go) calls `Deps.Restart` = `Recycler.Recycle`, which clicks save-and-exit and kills the
process. The `guard` goroutine of the running action (and `step`) polls `Health.Step`; it saw EventExited, latched
`ob.gone`, cancelled the action with cause `gameGone` and `do` called `fatalf("the game is gone (exited)")`. Log, 03:41:26
rung 4 -> `recycle: failsafe restart` -> 03:41:29 `fatal`. The bot killed its own game and then reported the death as
fatal. Only `betweenGoals` (idle recycle) cleared `ob.gone`; the failsafe path never did.
Fix: `expectGameExit(true/false)` brackets the restart; it ignores exit events meanwhile and forgets them afterwards.
Test: TestFailsafeRestartIsNotFatalAndGoalContinues (fails without the fix, V).
Run 1 (021327) ended differently: a step-2 reset, then the run was stopped at 03:26:48 ("shutting down: finishing in town",
operator or job end) and the game was already gone; same final lines but no rung 4. U: that job was killed externally.

## 1. Why no progress for 120 s in "clearing" (V logs, U root cause below)
Both stalls: `clearing`, `hostileNear 0`, position frozen, activity "Quest: fighting (Level up to 14)" (campaign goal).
- Run 2 (area 1, Rogue Encampment, i.e. the character is in town): the task executor ticks about once a second and
  every tick logs `quest.tools_of_the_trade/chores ... blocked ... ran but is still needed (evidence 2)`; the level step
  never runs (area 1 is a town, `LevelTo` returns nil for a town target, or the executor keeps picking the chores step).
  Nothing moves, nothing is killed, so the watchdog is right. U: the loop is in internal/task (not owned here): a blocked
  chores step must not be re-picked every tick, and a level step whose planner target is a town must pick another area.
- Run 1 (area 3, Cold Plains): `nav no exit known yet, exploring` from 3 to 9 (Cave Level 1) logged 6800 times
  (8 per second) with the character not moving: ExitTo gives up after 8 attempts, returns ErrNoExit, the next tick tries
  again. `LevelTo` avoids an area on "no exit to that level" but the log has no such warning, so U: the error did not reach
  it. Also "town: no town portal available" three times: the character had no TP scroll, so rungs 2 (reset) cannot work.

## 3. Changes (internal/bot only)
- Clearing own limit: `Config.ClearIdleSec` (default 60): no progress counter movement (step >= moveEpsilon, kill, damage,
  item, xp, area change) in the clearing state for 60 s is a stall; a real long fight or exploration always moves it, so a
  legitimately long clear is not flagged (TestClearIdleQuietWhileMoving). `StateStuckSec["clearing"]` below it still wins.
- Rung 1 now first calls `skipTarget`: for leveling and campaign goals the stalled target area is avoided 15 minutes and the
  planner re-plans. Farm goals name their area and are untouched.
- Rung 4 restarts the game and the login flow of the recycler; after success the ladder starts over, at most
  `Config.MaxFailsafeRestarts` (default 3) restarts without real progress in between, then the usual visible self-pause.
  The run never ends by the ladder in managed mode. Unmanaged (no Recycler) is unchanged: pause.
- Real exits not caused by the ladder stay fatal (TestFailsafeGameGoneStaysFatal).

## Not verified (needs the game)
Ready-to-run: soak i4 with a campaign goal on a character without TP scrolls; expect rung 1 at about 60 s, rung 4 restart,
OOG login, goal continues, no `fatal`. The fix for the task-layer loops above belongs to the task owner.

## 4. Follow-up: the two task-layer loops (agent/loopfix)
- Chores loop (V by reading + test): `Executor.handleFault` splices a Ready fault's Insert steps into `e.inserted`, which
  `Tick` always prepends to the plan nodes. When the spliced chores step kept failing (no gold, Need stays true) it was
  blocked (group blocked, planner moves to the level step) but stayed in `e.inserted`, so it was re-picked and blocked
  every tick and the level step never ran. Fix: `block`/`escape` drop the step from `e.inserted`; `handleFault` no longer
  splices a prerequisite that already failed MaxFails times (the faulting step takes the fail path). Test:
  TestBlockedInsertedStepDoesNotStarveLevelStep (12 chores runs and 0 level runs before, 3 and >0 after).
- ExitTo loop (U for the exact trigger, fix is defensive): ExitTo now counts explore rounds that claim success but leave
  position and known-cell count unchanged; the second one returns ErrNoExit. Any ErrNoExit from the exploring branch is
  remembered for 2 minutes per (from, to, known cells): a repeat call returns ErrNoExit at once, without logging or
  exploring, until the map grows. bot/task.go matches "no exit to that level" in the error text and avoids the area, so the
  planner picks another one. Test: TestExitToFruitlessExploreGivesUp (uses the `exploreFn` seam). Not reproduced in a sim:
  why done() was true while findGate failed is unknown.

## 5. Fourth fatal: game exits by itself after a chicken Save and Exit (agent/chickfatal)
Evidence (V, job livebot/20261029-035908-d2bot log, sentinel obs): 04:22:11 emergency town trip failed (no TP), life 63 ->
33 over 60 s with hostileNear 2 and the character frozen at (5247,5045); 04:23:07 `oog click save-and-exit` (the
chicken leave, no "chicken" log line); 04:23:08 area 0; 04:23:12 rung 1; 04:23:14 Game.exe gone (sentinel procAlive false,
nobody in the bot killed it: not rung 4) and `fatal: the game is gone (exited)`. U: why Game.exe died after the leave
(LeaveGame was V to keep it running before; i2/i3/i5 were also dead in the sentinel, so the box may be killing games).
Fix: `Runner.gameLost` (loop.go): an exit/crash/hang the bot did not plan restarts the game through `Deps.Restart`
(login included) in managed mode, `MaxFailsafeRestarts` times without real progress in between, then fatal as before;
unmanaged or a failed restart stays fatal. Tests: TestGameLostAfterChickenLeaveRestartsAndContinues,
TestGameLostRestartFailureIsFatal; TestFailsafeGameGoneStaysFatal now runs unmanaged.
`launch.Flow.BeforeSave` runs before the Save and Exit click; d2bot has no cheat Env yet, so install
`cheat.Env.GuardSave` there when one is attached (not wired, U).
Not fixed (U): the 60 s life drop. With 0 potions and no TP the character stood still in melee range (hostileNear 2,
no kiting log, position frozen 51 s) until chicken at roughly 20%. Needs live look: the clearing loop should walk to town
(or flee) once life is low and no potion/TP exists, instead of waiting for the chicken threshold.

## Stale exit after a planned restart (restartfix)

U (not reproduced on a quiet box, found by reading): the bracket forgot the latched exit, but an exit event the guard had not polled yet during the bracket (health is polled at most once a second) was read right after it and latched as an unplanned loss, so gameLost restarted again and spent the budget twice. expectGameExit(false) now reads and drops one pending health event. Both the rung-4 restart and gameLost use the same bracket, so each restart counts once.

## 6. Chicken / re-entry loop at low life (agent/chickloop)

Evidence (V, soak job livebot/20261029-050409-d2bot, 05:30-05:34, Booti clvl 12, 0 gold, no potion, no TP): every ~20 s
re-entry into the Rogue Encampment at 25 life, straight to "Quest: fighting" in Stony Field, trip "life low, cannot
regenerate, free heal" -> `town: no town portal available` -> abandoned -> chicken -> re-entry.

Causes found:
- V (log): the re-entry lands in TOWN, not in the field. The free heal there was blocked by the trip cooldown
  (`runGoal`: `now-lastTrip >= gap`, lastTrip set by the failed field trip), so she walked out unhealed.
- V (code): "cannot regenerate" is not misfiring. `resource.Input.LifeRegen` is never filled by the bot, so it is 0
  and the trigger fires for everyone. That matches D2, where life has no natural regeneration (S: game knowledge, not
  re-measured), so resting in the field cannot help; only the healer or potions can. The brief's "rest until life >=
  threshold" is therefore not implemented for life.
- V (code): `walkToTown` only handled Blood Moor; Stony Field (two hops) returned ErrNoPortal.

Changes:
- `bot/goal.go runGoal`: in town below the resume line (`Combat.EngageLifePct`, default 65) a heal-only trip runs
  regardless of the cooldown. `lowLifeGate` runs before target selection (also for task camps): in town it holds until
  healed (3 failed heals pause the goal), in the field with nothing hostile and no healing potion it calls ToTown;
  3 failed attempts pause the goal with a visible message. Never engages new packs below the line from the goal level.
- `town/trip.go waypointToTown` (act 1 only): no scroll and not next to town, current level has a waypoint -> take the
  waypoint to the Rogue Encampment. U: needs both waypoints activated, the town row click is the same panel as in
  BackToField; not run live. A level without a waypoint keeps ErrNoPortal (then the pause above).
- `bot/loop.go noteReentry`: 4 re-entries within 15 min pause the goal ("paused ... Look at the character").
- Tests: bot/chickloop_test.go (3), town TestToTownAndBack extended (needs /tmp/d2mpq).

Not done: combat-level PreEngage still only knows NearTown == Blood Moor; the fighter's own retreat is unchanged.

Live checklist: i4 with life < 65 percent, no scroll, in Stony Field: expect "no town portal: taking the waypoint to
town", heal at Akara, return; with the waypoint not active expect the pause after 3 tries.

## 7. Failsafe ownership (agent FAILOWN, strategy review T2)

Rule: for every fault ONE actor acts; every other actor that could see it is observe-only or structurally blocked.
Scope: the managed soak as `tools/soak/soak24.sh` starts it (d2bot --managed --supervise under tools/live.sh, no d2fleet,
d2soak as a read-only harness, d2sentinel a separate user service on the worker host). Tags: V = read in the code at the file:line
given (not run live), S = from docs, U = unverified. No row here has been exercised live except where a section above says so.

| fault | owner (acts) | where (V) | others, and what they do in a soak |
|---|---|---|---|
| death overlay | runner ladder (death branch) | internal/bot/loop.go:84 (`isDead` -> `deathWait`), death.go:47, Esc only at mode 17 death.go:69 | sentinel `death-overlay` (rules.go:434, 10 s, Esc): observe-only. d2soak counts deaths. |
| panel open, Esc menu open | runner ladder rung 1 (`CloseAll`, via no-progress) | loop.go:511, trigger observe.go:253 (`stuckCheck`); the bot has NO Esc-menu detector (V: no `EscMenu` in internal/bot, internal/combat, cmd/d2bot) | sentinel `esc-menu-stuck` 20 s / `panel-stuck` 90 s (rules.go:441, 448): observe-only. |
| OOG / menu screen | runner (`notInGame` -> `Cycler.EnterGame`) | loop.go:878, grace `MenuGraceSec` | sentinel `out-of-game` (rules.go:455) observe-only. `launch.Keeper` is not wired into d2bot (V: `Wire` has no caller outside launch), so it owns nothing. |
| game gone (exit, crash, hang) | runner `gameLost` -> `Deps.Restart` = `Recycler.Recycle` (the Recycler is the mechanism) | loop.go:406, observe.go:154 (`healthLocked`), cmd/d2bot/live.go:136; hang = `launch.Monitor` CPU ticks flat 45 s, monitor.go:93 | sentinel `proc-gone` 8 s / `game-hung` / `game-frozen` (rules.go:381, 416, 423) observe-only. Fleet `watch` (fleet.go:352) only under d2fleet. |
| helper gone | Recycler `Start` (`helper.Ensure`), reached by ladder rung 4 or `gameLost` | cmd/d2bot/recycle.go:90, loop.go:533 | sentinel `helper-gone` (rules.go:395) observe-only. Slow: input errors reach `fail` -> ladder (loop.go:391), so minutes (U). |
| frozen frames, busy game | runner (`Monitor` hung -> `gameLost`; a spinning game with moving CPU falls to the no-progress ladder) | monitor.go:91-93, loop.go:55-69 order: `gameEvent` before `stuckCheck` | sentinel `game-frozen` (needs frame AND CPU flat 90 s) observe-only; tools/devhealth.sh prints frames, never acts. |
| no progress | runner ladder (rungs 1 skip target + Esc, 2 town reset, 3 leave/re-enter, 4 restart, then visible self-pause) | observe.go:253, loop.go:477; per-state limits 60 to 300 s, global `GlobalStuckSec` 600 (config.go:41,88) | sentinel `no-progress` 480 s (rules.go:501) observe-only; d2soak `StallXP` 10 min alert (status.go:29). |
| panic | runner `safeStep` recover + budget (self-pause, then `ErrPanicStorm`); other goroutines `goSafe` | bot/panicsafe.go:28, cmd/d2bot/safego.go:23 | budget exhausted ends the process: that is the next row. |
| d2bot process dead (panic storm exit 4, any non-zero exit, watchdog exit) | supervise (`d2bot --supervise`): backoff, 10 per hour, `--clean-start` stops the leftover game | cmd/d2bot/supervise.go:68, game.go:67; exit 3 (deliberate fatal) is NOT restarted | under d2fleet `ProcBots.Run` instead (bot.go) and `--supervise` is ignored (main.go:174). Sentinel `bot-silent`/`bot-stuck` need a `botUrl` and `botRestartCmd` in a sentinel config: none ships (V: no tools/sentinel.json), so they never see a bot. |
| deadlock, stuck tick | TICKWATCH watchdog: no finished loop iteration for 10 min (`D2BOT_WATCHDOG_MIN`, 0 = off) dumps stacks and exits 4 | cmd/d2bot/watchdog.go:61, live.go:197, loop.go:23,40 | its exit is handled by supervise (row above); it never restarts anything itself. |
| heap runaway (OOM) | HEAP GUARD (`heapGuard`, knob `bot.heapLimitMB`, default 4096, 0 = off): heap in use above the limit (checked every 2 s) logs `watchdog: heap N MB > limit: writing heap profile and exiting for a restart`, writes `heap-<ts>.pprof` and `goroutines-<ts>.txt` to `<state>/heapguard/` (newest 3 kept) and exits 4; a GC soft limit of limit+25 % is set at start (skipped when GOMEMLIMIT is set) | cmd/d2bot/heapguard.go, live.go (next to the TICKWATCH goroutine) | exit 4 is restarted by supervise (row above); d2soak `heap_runaway` (crit: bot RSS > 8 GB or +1 GB in 5 min) only reports; a systemd `MemoryMax` on the bot (docs/soak24.md section 8) is the last net: it kills the bot, never the host. |
| low-life stall | runner goal level (`lowLifeGate`, free heal, town trip, then goal pause) | bot/goal.go:839, section 6 above | none; d2soak reports deaths/hour. |
| wine / process leak, fd growth | Recycler policy (cycles, fd count) at `betweenGoals`; orphan reaping on every stop (`reapAfterStop`) | launch/recycle.go:103, bot/loop.go:118, launch/reap.go:189 (calls game.go:71, lifecycle.go:411) | d2soak `wineAlert` and fd slope alert are observe-only; sentinel `idle-reaper` does not apply (needs no bot and no lock). |
| fleet slot restart | fleet `supervise` (per slot, backoff, `MaxRestarts` 10 in d2fleet) | fleet/fleet.go:221, cmd/d2fleet/main.go:221 | not part of soak24 (no fleet). Under a fleet the sentinel must stay observe-only (see overlap F3). |
| soak harness | d2soak: observe only (V: no input package in cmd/d2soak; soak.sh only systemd-runs d2soak; the only os/exec is the optional alert hook D2SOAK_ALERT_CMD, unset by default, docs/soak-analysis.md section 13: it can run a user-chosen notification command and never restarts or kills anything) | cmd/d2soak, tools/soak/soak.sh:71 | it alerts and writes alerts.json, nothing else. |

Gate refusals (`errAreaGated`: banned or gear-gated route) never feed the ladder: `do` handles them before `fail`
(see deathspiral.md, "Area ban is a planning block").

### 7b. Heap runaway (agent NAVMEM, 2026-10-30)

G1 attempt 16 on the worker host: the Linux OOM killer ended d2bot (anon-rss 48.8 GB, 04:45:45) after the bot log went silent at 04:42:40 (V: soak log and dmesg quoted in the brief). The d2soak samples show the heap went from 43 MB (04:42:25) to 27.8 GB heap in use / 32.5 GB RSS (04:42:55) with 31 goroutines, i.e. one goroutine allocating about 1 GB/s of LIVE memory inside one call (V: `i4-20261030-030625/samples.jsonl`). Before this the heap had been flat for 14 h. The guard (V offline tests in cmd/d2bot/heapguard_test.go, live U) turns the next such event into a profile plus a normal supervised restart within 2 s. Root cause and the nav fixes: docs/nav.md "NAVMEM".

### Overlaps: where two actors would both act, and the race
S1. Frozen or dead game, sentinel acting + ladder/`gameLost` + watchdog + supervise (the case in the brief). Timeline for a game
that stops responding: bot Monitor hung at 45 s -> `gameLost` -> `Recycle` (Leave fails, SIGTERM, SIGKILL, launch, login: up to
`restartTimeout` = 3 x `TownTimeoutSec` 180 = 540 s per attempt, two attempts with a 20 s pause, loop.go:406-440, V by
arithmetic); sentinel `game-frozen` at 90 s, `proc-gone` 8 s after any exit; watchdog 10 min; supervise only after d2bot dies.
Race 1 (two restarters of one game): the sentinel's restart/OOG/leave/stop take the instance flock (sentinel/act.go:93-101,
:44; LockWait 20 s) and a run started by tools/live.sh holds that lock for its whole life (live.sh:52,58), so under soak24 the
sentinel's restart is `skipped`. BUT d2bot itself never takes the lock (V: `LockInstance` callers are only cmd/oog, cmd/d2game
and the sentinel). Started any other way (by hand, or by d2fleet ProcBots, which sets no flock) a sentinel in `--act` mode and
the bot's `Recycle` would both stop and launch Game.exe: a double launch or a SIGKILL of the game the bot just started. The
lock is the only thing separating them in soak24, so the sentinel must not act at all (config below), not only be locked out.
Race 2 (Esc class, no lock): death-overlay, esc-menu-stuck, panel-stuck and no-progress step 1 press Esc without the lock
(act.go:93-101 lists only four locked actions). Esc with nothing open OPENS the game menu (docs/oog.md), so a sentinel Esc
landing right after the bot's own respawn or `CloseAll` can leave the Esc menu open, which the bot cannot detect (table row 2)
until its no-progress timer fires. The sentinel verifies from memory before pressing (act.go esc), which narrows but does not
close the window (U).
S2. Watchdog vs a long legitimate step. `lastStep` is stored once per loop iteration (loop.go:23,40) and a restart runs inside
one step. `gameLost` with a failed first attempt is 540 + 20 + up to 540 s = 1100 s, above the 10 min watchdog (600 s): the
watchdog kills d2bot in the middle of the second launch. Consequence is bounded and self-healing (exit 4, supervise restarts
with `--clean-start`, which stops the half-started game, game.go:66), but it counts twice against supervise's 10/hour and
resets the bot's in-process restart budgets. Not fixed: raising the watchdog is a default change the brief excludes for soaks
(watchdog stays default); the real fix is code (touch `lastStep` inside `Recycle`/`gameLost`), recorded as a follow-up.
S3. supervise + watchdog + fleet ProcBots: `--supervise` is skipped under d2fleet (main.go:174, `D2BOT_FLEET=1` or parent comm
`d2fleet`), so the two never nest (V). The other direction differs: `ProcBots.once` restarts d2bot on ANY exit including 3
(deliberate fatal) up to `MaxRestarts` 10 consecutive with backoff to 2 min (fleet/bot.go:332-350, d2fleet/main.go:213), while
supervise stops at 3 (supervise.go:75). A corrupt character therefore loops 10 times under a fleet. Code change, not done.
S4. Fleet `watch` vs d2bot restarts (fleet.go:352-380): for a bot-owned game the fleet gives d2bot `GameWait` (3 min) to bring
a missing Game.exe back, then stops bot and game itself. A d2bot restart that has the game absent (Shutdown 15 s plus
Launch up to 90 s plus helper) normally fits, on a loaded box it may not (U); then both the fleet and d2bot's `Recycle` act on
one slot. Fleet only, not soak24; the safe default (`GameWait` above the `Launch` timeout) is already met, the margin is thin.
S5. Ladder rung 4 + `gameLost` + `betweenGoals` share one hourly cap (`restartCapHit`, loop.go:459) but have separate
per-progress budgets (`restarts`, `goneRestarts`). All three run on the runner goroutine, so they never overlap each other (V:
single `step` loop); a failsafe restart cannot collide with a scheduled recycle.
S6. d2soak and sentinel observing the same instance: no race, both read-only in soak. d2soak's `TickAge` and restart alerts use
the bot log, not the sentinel.

### Soak configuration that matches the table (`tools/soak/soak24.sh`)
- Sentinel: not started by soak24 (it is a separate user service). Preflight asks the worker host for `/api/v1/sentinel` mode (loopback,
  no token) and FAILS when any rule may act on the soak instance (`act:all`, `act:<rules>`, `...@<this instance>`); `observe-only`
  or "not running" pass. Fix: `tools/sentinel.sh start` (observe-only is its default) or `tools/sentinel.sh stop`.
- supervise: ON (`--supervise`, `--supervise-max` default 10/h). Watchdog: default 10 min; `live.sh` forwards no environment,
  so `D2BOT_WATCHDOG_MIN`, `D2BOT_SUPERVISE` and the recycle variables cannot leak into a soak (V: live.sh:52 `env D2BOT_INSTANCE=` only).
- Preflight and `start` print the effective failsafe config (tools/soak/failsafe.sh); test: `bash tools/soak/failsafe_test.sh`.

Not verified (needs the game): that a sentinel in `observe-only` really stays quiet next to a running bot, the Esc race, and
S2's 1100 s case. Ready-to-run: with the sentinel observe-only, soak24 `--gate` on i4, kill Game.exe at ~60 min and read
`sentinel.jsonl` (`would` lines only, no `acted`) and the bot log (one `game lost, restarting it`, one `recycle done`).

### FAILFIX follow-ups (S1-S4 fixed in code)
- S1 (V: unit tests, no live): d2bot's `Recycler.Guard` takes `launch.LockInstance(id, 0)` (non-blocking, same files as live.sh and the
  sentinel) for the whole recycle; when busy it logs "instance lock busy" and the recycle fails (gameLost retries once after 20 s).
  Guard is off when the lock is already held at d2bot start (a live.sh wrapper owns it; our own open file description would
  otherwise conflict with it). Caveat (U): a sentinel holding the lock for a moment exactly at d2bot start disables the guard.
- S2 (V: unit tests): `Runner.Beat()` stamps the watchdog heartbeat; `Recycler.Progress` calls it after each finished phase
  (leave/stop/start/login), `gameLost` after each restart attempt and the retry pause, `oogRun` on entry and exit. No timer: a
  phase that hangs still trips the 10 min watchdog.
- S3 (V: unit test): `ProcBots.Run` treats an exit code 3 as final (slot `failed`, no restart).
- S4: fleet `GameWait` default 3 -> 6 min.

### 7a. No-area in recovery is not a ladder input (DEATHCHAIN)
`recoveryStep` wraps LevelTo's "no eligible reachable leveling area" in `errAreaGated`; `r.do` treats it like AREABAN's refusals (failures reset, gatedWait, no `escalate`). G1 attempt 11 climbed rungs 1-4 in 16 s on this error and recycled the game at 22:13 (V, log).
NOIDLE (G1 attempt 14): a stall with a "Waiting: ... banned" activity in town is not a ladder input either. `escalate` first calls `idleBreak(true)` (lifts the least bad area ban, farms it); nav-stuck skips are 150 s (`bot.navSkipSec`), not 15 min. See docs/deathspiral.md "NOIDLE".

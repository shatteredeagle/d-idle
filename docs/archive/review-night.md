# Review: night merges (0ba00fe..b068c2a)

REVIEW4, 2026-10-29. Scope: PANICSAFE, QUESTEVENT + NMSIM merge, emit() direct push, LOOTPOTION, DOCTOR, SEALSIM.
Docs only, no code changes. Tags: **V** = read in the repo at b068c2a or run (scratch copy, tests below);
**S** = reasoned from code, not run; **U** = unknown / needs live.

Scratch reproductions (in a copy outside the repo, not committed): `go test -p 1 -run Review4 ./internal/goals/ ./cmd/d2bot/`.
All three fail on master, which confirms H1, H2 and H3.

## Summary table

| # | Sev | Area | One line |
|---|-----|------|----------|
| H1 | High | goals merge | `m.locked` is never cleared on level-up resume / ResumeGoal / Remove; a stale lock auto-resumes a goal the user paused (V, test) |
| H2 | High | supervise | `--supervise` restart re-sends `--goals campaign`: every crash queues one more campaign goal (V) |
| H3 | High | supervise | `D2BOT_SUPERVISE=1` in env: the child inherits it and supervises again, an unbounded chain of supervisors (V, test) |
| M1 | Med | panicsafe | over-budget panic with no active goal or already self-paused: selfPause is a no-op, loop panics every ~10 s forever, never exits for the supervisor (V read) |
| M2 | Med | supervise | deliberate fatal halts (exit 1) are restarted with `--autostart`, so the same fatal recurs up to 10x/h and then the soak dies anyway (V read) |
| M3 | Med | safego | goSafe gives up on the manager after 10 panics/h and the process stays up with no scheduler; nothing reports it (V read) |
| M4 | Med | panicsafe | recover() over manual Lock/Unlock pairs: a panic between them turns a crash into a permanent deadlock the supervisor cannot see (S) |
| M5 | Med | panicsafe | a guard panic leaves the running action unguarded (no death/menu/stuck cancel) until its timeout (V read) |
| M6 | Med | clean-start | `GameShutdownWait` (4 s) for the clean-start kill of a Wine game; failure exits 1, and a surviving old game gets adopted by a non-ancestor d2bot (S/U) |
| M7 | Med | HeldAt | `Goal.HeldAt` is written and cleared but never read: dead persisted state, and NMSIM's "persist hold across restart" now relies only on `Locks` (V) |
| L1..L9 | Low | various | see below |

## High

### H1. Stale locks: the merge dropped both `delete(m.locked, ...)` calls (V, test)
- `internal/goals/manager.go:469-479` (unlockLocked) gates on `m.locked[g.ID]` (QUESTEVENT) but clears only `g.HeldAt`
  (NMSIM); `manager.go:903-918` (ResumeGoal) likewise. Before either branch both functions deleted the lock. Only
  `questLocked` (`manager.go:507`) still deletes. `Remove`/prune never did. `Locks` is persisted (`store.go:30`,
  `manager.go:193`), so stale entries survive restarts.
- Failure: goal held at clvl 30 -> level 31 resumes it (lock {30} stays) -> user pauses it from the dashboard ->
  level 32 arrives -> `unlockLocked` sees a lock, state paused, 32 > 30 -> resumes and runs the goal the user paused.
  The same happens with a quest Done event for a lock carrying an Unlock flag. Scratch test
  `TestReview4StaleLockOverridesUserPause`: "user pause overridden by level-up: state active";
  `TestReview4ResumeGoalLeavesLock`: lock left by ResumeGoal and by Remove.
- Fix: pick ONE source of truth. Simplest: keep `m.locked` (it carries Unlock/Diff), `delete(m.locked, g.ID)` in
  unlockLocked, ResumeGoal, PauseGoal (a user pause replaces a hold), Remove, Reset and prune; drop `Goal.HeldAt` (M7),
  or rebuild `m.locked` from HeldAt in restore and stop persisting Locks. Also clear the lock on any non-paused
  state in `restore` (self-heal of the files already written). Add the two scratch tests to manager_test.go.

### H2. Supervised restarts duplicate the `--goals` queue (V)
- `cmd/d2bot/supervise.go:222-248` (childArgs) keeps every flag but `--supervise*`, `--run-for`, `--clean-start`.
  `cmd/d2bot/live.go:152` calls `enqueueGoals(m, o.goalSpec)` on every start, and `addAtLocked`
  (`internal/goals/manager.go:708`) does not dedupe. soak24 starts with `--goals campaign --autostart`
  (`tools/soak/soak24.sh:96`).
- Failure: first crash in the 24 h soak -> the child loads goals.json (campaign g1 in progress) AND appends a second
  campaign g2. With chaining, g1 completes Normal and chains Nightmare, while g2 (Normal) sits in the queue: the
  order depends on the priority/insert logic, and in the worst case the bot goes back to a Normal campaign or runs the
  quest chain twice (U which). Each further crash adds another one.
- Fix: on a restart (attempt > 0), childArgs must strip `--goals` (and `D2BOT_GOALS` from the child env) and keep
  `--autostart`, the same rule fleet.ProcBots already applies (`internal/fleet/bot.go:150-172`). Test in supervise_test.go.

### H3. `D2BOT_SUPERVISE` env makes every child a supervisor (V, test)
- `cmd/d2bot/main.go:87` defaults `--supervise` from `D2BOT_SUPERVISE`; `runChild` (`supervise.go:253`) uses
  `exec.Command` with the inherited environment; childArgs only strips the flag. Scratch
  `TestReview4EnvSuperviseRecurses`: child options have `supervise=true`.
- Failure: anyone (a systemd unit, a fleet Spec env, a container env) setting `D2BOT_SUPERVISE=1` gets
  supervisor -> supervisor -> supervisor ... each spawning the next immediately, until pid/memory limits; no bot runs.
- Fix: in runChild set `cmd.Env` to `os.Environ()` minus `D2BOT_SUPERVISE` (plus `D2BOT_GOALS` on restarts, H2), or
  have the child refuse `--supervise` when a marker env (`D2BOT_SUPERVISED=1`) is set. Test it.

## Medium

### M1. Panic budget does not escalate when there is no goal or the bot is already self-paused (V read)
- `internal/bot/panicsafe.go:67-70` calls `selfPause`, which returns without doing anything when `r.goal == nil` or
  `r.selfStop` (`internal/bot/loop.go:570-575`). Then only `pause(ctx, 10s)`.
- Failure: a deterministic panic in code that runs every tick regardless of the goal (observe/pollQuests, applySettings,
  betweenGoals, the self-paused step path) -> panic every tick+10 s forever, each logging a full stack (~a few KB:
  order of 30-50 MB/day of log) and emitting an EvError into the manager ring and the bus. The process never exits, so
  `--supervise`, whose clean restart is the one thing that would clear corrupt in-memory state, never triggers.
- Fix: count budget overruns; on the second overrun within PanicWindow (or any overrun with no goal) return a fatal
  from Run (exit non-zero, distinct code) so the supervisor restarts the process. Rate-limit the stack log (full stack
  once per distinct panic value, then counts).

### M2. Supervisor restarts deliberate fatals and re-applies `--autostart` (V read)
- `supervise.go:196` restarts any non-zero code. `run` returns `bot halted: <fatal>` -> exit 1 (`main.go:161-164`);
  the manager's `haltLocked` (mode stopped) is persisted, but the child is restarted with `--autostart`/`--goals`,
  which calls `m.Start()` (`live.go:158`) and overrides the halt.
- Failure: a real fatal (corrupt character, unusable config, port in use) -> 10 restarts in roughly 30 min (backoff
  5 s..2 min plus about 1-2 min of game launch each), each killing and relaunching Wine, then the supervisor gives up
  and the soak ends anyway, after churning the game ten times. Parse errors exit 2, the same code as an unrecovered Go
  panic.
- Fix: reserve exit codes: 0 clean, 3 = deliberate fatal (do not restart), 2 / 128+n / 1 = crash (restart). Have
  `run` return the fatal code for `bot halted` caused by `ev.Fatal`.

### M3. goSafe giving up leaves a zombie process (V read)
- `cmd/d2bot/safego.go:108-111` returns after 10 panics/h; the comment says "the health check shows it", but nothing
  records it (no registry/health field; grep finds none). For "manager" that means no Handle and no Tick: the runner
  finishes its goal and idles forever while /health looks fine. For "runner", goSafe returning makes runnerDone fire
  with `err == nil`, and `fmt.Errorf("bot halted: %w", nil)` yields "bot halted: %!w(<nil>)" (works by accident, ugly).
- Fix: for runner and manager, giving up must end the process non-zero (cancel the top ctx with a cause), so the
  supervisor restarts it. Expose per-goroutine panic counts in Health.

### M4. recover() can turn a crash into a silent deadlock (S)
- Many critical sections use manual `Lock()/Unlock()` without defer (`internal/bot/observe.go:180-191`
  expectGameExit calls `r.d.Health.Step` inside obMu; `loop.go:72-75, 124-126, 528-537, 780-791`; `task.go` several).
  Before PANICSAFE a panic there killed the process (visible, restartable). Now safeStep / the guard recover and the
  next `obMu.Lock()` blocks forever: the runner goroutine and every dashboard read of the snapshot (`observe.go:740`)
  hang; the process stays up, so the supervisor never restarts it.
- Fix: `defer` the unlock in any section that calls out of the package (Health.Step, readers, emit); in the step
  watchdog / soak, alert on "runner tick age > N min" (or have Health() report the last step time) and exit non-zero
  from a watchdog goroutine when the loop stops ticking.

### M5. Guard panic leaves the action unguarded (V read)
- `internal/bot/loop.go:299-306`: the guard's recover only counts the panic. The guard is what cancels on death,
  menu exit, game gone, and pause (`loop.go:732-765`). The action keeps running until its step timeout.
- Failure: the guard panics on a bad snapshot during a fight -> the fight continues after death or after the game
  dropped to the menus, clicking blindly until the timeout.
- Fix: in that recover, call `cancel(errGuardPanic)` so the action ends and takes the normal failure path.

### M6. Clean-start robustness (S/U)
- `cmd/d2bot/game.go:67-73` kills the leftover game with `bot.GameShutdownWait` (4 s, `internal/bot/shutdown.go:14`).
  Wine often needs longer; an error returns `clean start: ...` -> exit 1 -> another restart (counts against the 10/h).
  If Shutdown "succeeds" but `game.Find()` still sees the old process, the launch is skipped and the new d2bot adopts a
  game it is not the ancestor of: with `soak24 --nocap` the Yama ptrace_scope=1 model breaks and memory reads fail
  (U, docs/soak24.md says the clean-start path is untested live).
- Fix: use a longer clean-start grace (about 20 s) with a SIGKILL fallback, re-check Find() after the stop and fail
  loudly (distinct code) if the old game is still there; run one deliberate-crash drill on a throwaway instance before
  the 24 h soak (kill -SEGV the child, watch the relaunch and the re-login).

### M7. `Goal.HeldAt` is dead state (V)
- Written at `manager.go:301`, zeroed at 475/709/914/948, never read (grep: no other use, no test). The merge message
  says "keeps NMSIM HeldAt persistence", but the restart path now works only through `fileState.Locks`. Two sources
  that can disagree (HeldAt=0 with a lock present after ResumeGoal, and the reverse for a state file written by an
  NMSIM-only build: paused goal, HeldAt set, no Locks -> never auto-resumes).
- Fix: see H1 (one source of truth). If HeldAt stays, migrate it in `restore` into `m.locked` when Locks is absent.

## Low

- **L1 emit ordering (S).** Hold/EvQuest go straight to `r.push` (`internal/bot/runner.go:250-257`) while LevelUp goes
  bus -> bridge goroutine -> push. The Hold's `Level` is `snap.Player.Level` from the start of the farm loop
  (`farm.go:121`), which can be stale after a long loop: a level-up already handled, then a Hold at the old level, means a
  one-level delay before the next resume. No deadlock found: `push` never blocks, and `emit` is never called with
  `r.mu` held (notePanic and selfPause unlock first); pollQuests emits with obMu held, which is fine because push and
  bus.Publish take no runner locks. Fix: read the current level when the Hold is emitted.
- **L2 two producers on `r.ch` (S).** The bridge and runner goroutines now both push; drop-oldest (cap 2048) can drop a
  Hold only if the manager stalls (a slow `saveLocked`) under an event storm. Acceptable; a priority path for Hold/Quest
  would remove it.
- **L3 repeat quest events save the file (V).** `manager.go:278-281` calls `questLocked`, which always calls
  `saveLocked` (`manager.go:515`) even for repeats. After every restart the runner re-emits every non-empty quest once,
  so about 27 synchronous writes in a burst. Fix: save only when something resumed or the questSeen entry changed.
- **L4 pollQuests under obMu (V).** `observe.go:756-789`: a save stat (+ read on change) and memory reads every 15 s with
  obMu held, on both the runner and the guard goroutine (poll is called from both). Cheap with the mtime cache; move it
  out of the lock if reads ever get slow.
- **L5 orphaned child (S).** If the supervisor is SIGKILLed alone (`jobs.sh kill` signals the session, so normally both
  are hit), the child and its game keep running unsupervised. Add `Pdeathsig: SIGTERM` on the child from a locked
  thread (the trick in `internal/launch/proton.go:58-90`). SIGTERM handling itself is fine: `soak24 stop`'s
  `pkill -f` hits both supervisor and child; the child's handler absorbs the second TERM; exit 0 ends supervision.
- **L6 fleet interplay (S).** Nothing stops a fleet Spec from passing `--supervise` (or the env, H3). Then ProcBots
  (StopGrace 70 s, `internal/fleet/bot.go:74`) and the supervisor (100 s grace) nest: ProcBots SIGKILLs the supervisor
  first, the child is orphaned (L5), and ProcBots starts a second d2bot on the same game. Fix: d2bot refuses
  `--supervise` when started by d2fleet (env marker), or ProcBots strips it. The Recycler is in-process, so a crash in the
  middle of a recycle is covered by clean-start (M6 caveats apply).
- **L7 DOCTOR (V).** No check writes anything: config/token checks only read and stat; the token check prints file names
  and modes, never contents (`internal/setup/doctor.go:98-123`); helper.reach is a bare connect+close, the same as the
  existing `launch.pingHelper`; char.difficulty only attaches read-only via `mem.OpenInstance`. Nits: a symlink in
  tokens/ is judged by its lstat mode (false fail); the legacy `~/.config/<name>/token` path and the tokens dir mode are
  not checked; instance.paths assumes slots are i0..iN-1; GET /checks (`setup/http.go:50`) exposes data paths and token
  file names to whoever can reach the wizard API (keep it on localhost).
- **L8 LOOTPOTION (V).** `Looter.build` is written in `applyClass` from `behaviors.prepare`, on the runner goroutine,
  the same goroutine as pickup, so there is no race. If the first snapshot fails, the build stays "" and NM/Hell pickup
  falls back to `PlanForDiff`, so the shop and pickup can disagree until the next prepare (minor). The Normal path is
  unchanged (`town/belt/belt.go:263-268`).
- **L9 SEALSIM (V).** The sleeps after a failing Attack (`boss/seals.go:352`, `boss/waves.go:176`) fix a real hot loop;
  `openSeal`'s death check (`seals.go:315`) and `passFatal` in OpenSeals are correct. If `w.View` fails after the clicks,
  the error is dropped and the result is still ErrSealStuck. Fine.

## What the 2 h gate and the 24 h soak hit first

- **2 h gate (S).** Most likely nothing from this batch unless the bot crashes. If it does, the first restart is also the
  first live run of clean-start (M6) and immediately queues a second campaign (H2). A deterministic panic would show as
  a self-pause cycle or the M1 spin rather than a restart; watch the logs for `panic recovered`, since soak alerts do
  not read `Health().panics` (nothing in tools/soak or cmd/d2soak reads it).
- **24 h soak (S).** In order of likelihood: (1) H2 duplicate campaign goals after the first crash; (2) M2/M6 restart
  churn if a crash repeats (10/h cap, then the soak ends); (3) M1/M4 a process that is alive but stuck, which the
  supervisor cannot see; (4) Hold/resume churn every level-up for a clvl-locked run (by design, but noisy). H1 needs a
  manual pause, so it hits interactive use and the dashboard rather than the soak.

## Recommended actions (in order)
1. H1: one lock source of truth, delete locks in all resume/pause/remove paths, self-heal in restore, add the tests.
2. H2 + H3: childArgs/runChild strip `--goals`/`D2BOT_GOALS` on restarts and `D2BOT_SUPERVISE` always; add tests.
3. M2: exit-code contract (3 = deliberate fatal, do not restart).
4. M1/M3/M4: escalate to a process exit when panics persist, goSafe gives up, or the loop stops ticking.
5. M5: cancel the action on a guard panic.
6. M6: one crash drill on a throwaway instance before the 24 h soak; longer clean-start grace.

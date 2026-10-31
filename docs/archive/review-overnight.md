# Merge-risk review: overnight / 24 h soak (master a5ff398)

Reviewer: REVIEW (Opus), 2026-10-29. Range `54253e4~30..master` (142 commits), focus on the failsafe and restart code
in internal/bot, cheat god undo, the task executor drop-blocked fix, resource.Flee/PovertyCheck and the planner floor.
Docs only, no code changed. Tags: **V** = read in the repo or run here; **S** = reasoned from the code, not run;
**U** = needs a live check.

Run here (V): `go test -p 1 -count=1 ./internal/task/ ./internal/resource/ ./internal/cheat/ ./internal/bot/`, all ok.
The tests use a fake cycler that ignores ctx and returns at once (internal/bot/runner_test.go:755), so they cannot catch
H2 below.

No blocker found. Three highs can make a 24 h soak spin without progress or stop by itself. None of them can lose a character
(softcore, throwaway characters). M2 can quietly corrupt a throwaway character's max life.

## High

### H1. Restart and ladder budgets are reset by gold, potions and damage, so they do not bound a crash loop
- V `internal/bot/observe.go:438-441`: any gold change calls `bumpReal`, *before* the in-town early return, so buying
  potions, repairing or selling in town counts as "real progress". `:567-568` a drunk potion and `:479-481` monster life
  falling count too.
- V `internal/bot/loop.go:374-376` (gameLost) and `:402-404` (escalate): any change of `ob.real` sets `goneRestarts`,
  `restarts` and `stuckStage` back to 0.
- V `gameLost` never calls `noteReentry` (`loop.go:739` is the only caller), so the "4 re-entries in 15 min, pause" breaker
  from chickloop does not see restarts after a game exit. The comment at `loop.go:366` names this as the main case: a chicken
  Save and Exit after which Game.exe exits.
- Result (S): crash, restart, town chores (gold changes), walk out, crash again. Or chicken, game exits, restart, chicken
  again (monster damage and potions count as real). `MaxFailsafeRestarts=3` never trips, so the bot loops all night with
  no visible error. A crash that comes back every time (one area under Wine, a chicken spot) spins for 24 h and spends the
  potion gold on every trip.
- Fix: count only XP, level-up and quest progress as `real` (keep gold, potions and damage as watchdog `bump`), or reset the
  budgets on an XP gain of at least N. Call `noteReentry` from `gameLost`, and add a wall-clock cap such as
  "at most K unplanned restarts per hour", then selfPause.

### H2. Rungs 3 and 4 and the menu re-entry run under the guard, which cancels them after 8 s at the menus
- V `loop.go:645-649`: the guard cancels any action with `errLeftGame` once the observer has been at the menus for
  `MenuGraceSec` (default 8). `errLeftGame` is in the "ignore" list at `:336`, so it is never counted as a failure.
- V `Flow.Start` checks `ctx.Err()` on every pass (`internal/launch/oog.go:362`).
- `notInGame` (`loop.go:735`) only runs once the menu grace has expired, so (S) the guard cancels `EnterGame` on its first
  tick (WatchMs 50). Each bot step then gets roughly one click of the OOG flow ("works by slicing"). The flow's local
  state (`created`, `leaves`, `ipTyped`, screen timers) is rebuilt on every call.
- Rung 4 (`loop.go:446-461`) runs `Recycle` inside `r.do`. After the relaunch the new game sits at the main menu for more
  than 8 s, so the guard cancels the Login part (S). `ok` is false, the restart is still counted, and `stuckStage` stays 4.
  If there is no real progress before the next stall, the next escalation goes straight to the self-pause. The restart
  rung has not been verified live since the expectGameExit fix (`docs/stallfatal.md:41`, "Ready-to-run").
- Rung 3 (NewGame) and `ensureDifficulty`/`noTarget`/`freshGameToLevel` NewGame hit the same check when the menus take
  more than 8 s (loaded box).
- Fix: suppress the menu-grace cancel while the running action is itself an OOG action. For example, pass a flag
  through `runAction`, or set an `ob.oogAction` bracket like `expectGameExit`. Add a test whose fake cycler honours ctx and
  spends more than MenuGraceSec at StateMenu.

### H3. Task executor counters never reset within a goal; `givenUp` is permanent and misses timeouts and deaths
- V `internal/task/executor.go:120` `Reset` has no caller in internal/bot (grep). A new executor is only made on a new goal
  generation (`internal/bot/task.go:144`). So `fails`, `blocked` and `skipped` live for the whole campaign goal, in a 24 h
  soak too, across new games and gameLost restarts.
- V `executor.go:363-371`: once a prerequisite (for example Chores with no gold) reaches `MaxFails`, it is never spliced in
  again for that goal. Its parent takes the fail path and blocks its group. When every group is blocked, `plan.Stuck` leads to
  `selfPause` (`bot/task.go:195`). A short poverty spell early in the night can therefore become a permanent pause.
- V `givenUp` looks only at `e.fails`. An inserted step that escaped through the timeout path (`:347-353`, counts
  `timeouts`) or was blocked by deaths (`:338-341`) is dropped by `dropInserted` but spliced in again on the next Ready
  fault. That is the starvation loop f6acda0 fixed, reopened through another path (S: bounded only by the group block if
  the planner honours it).
- Fix: call `ex.Reset()` (or a softer `ForgetFails`) when `syncTaskGame` sees a new game and after `gameLost`, or let
  failures decay with time. Make `givenUp` also check `timeouts > MaxTimeouts` and `deaths >= MaxDeaths`.

## Medium

### M1. One failed unplanned restart is fatal; the rung-4 timeout is shorter than the recycle can take
- V `loop.go:390-392`: any `Restart` error ends the run (`fatalf`). There is no retry or backoff for a transient
  failure such as a Launch timeout or OOG `ErrStuck`. Nothing restarts d2bot overnight (S; sentinel not checked).
- V rung 4 uses `sec(TownTimeoutSec)` = 180 s (`loop.go:454`), but `Recycle` = Leave (30 s, `cmd/d2bot/recycle.go:86`) +
  Shutdown (15 s) + helper Ensure (60 s) + Launch (90 s) + Login can exceed 195 s on a loaded box. Then errStepTimeout
  counts it as a failed rung and the restart slot is spent. gameLost uses 3x (540 s), which is inconsistent.
- Fix: one retry with a pause before going fatal. Use the same restart timeout for both paths, larger than the sum of the
  recycle steps.

### M2. A stale god marker is restored without checking that life is inflated; the restore writes merged into base
- V `internal/cheat/godstate.go:104-113`: when a marker for this character exists, `pendingRestore` writes `m.MaxLife`
  even if the current max life is normal. Scenario: a god job is SIGKILLed and the game is killed before any save, so the
  `.d2s` is clean but the marker stays. The character levels without god. The next god start, `d2cheat restore` or
  `GuardSave` then lowers max life to the old value, and the game saves it.
- U `restoreLife` writes the value read from `VecMerged` into both vectors (`setBoth`, `memory_ops.go:108`). If the merged
  vector includes gear or Battle Orders bonuses, each god cycle bakes those bonuses into base max life. The Barbarian is
  now the default build, so BO is in play. The same pattern existed before this range.
- Fix: restore from the marker only when current max life is above `godSaneMax` or equals `godBuffer`, and otherwise
  delete the marker as stale with a ledger line. Verify the base vs merged semantics live on a geared throwaway.
- L: `godSaneMax` (6000 life) can be passed by a high-level BO Barbarian, which would refuse god with ErrInflatedLife.

### M3. GuardSave is not wired into d2bot; a code comment says it is
- V there are no assignments to `Flow.BeforeSave` outside tests. `oog.go:183-184` says "The bot installs
  cheat.Env.GuardSave here", which is false (docs/cheats.md:191 and docs/stallfatal.md:68 correctly say it is not wired).
- d2bot does not run god, but a dev rig (act2live, navlive, townlive, combatlive) killed on the same instance and
  character leaves an inflated life. An overnight d2bot run then saves it on every area change and on recycle.
- Fix: at d2bot start, call `PendingGod`/`pendingRestore` and refuse or repair before the first goal. Fix the comment.

### M4. The planner floor counts avoided areas as "excluded window", and the fallback message is wrong
- V `internal/bot/plan.go` `windowExcluded` treats any `Exclude` as covering the level. `target0` (`goal.go:424-436`) puts
  15-minute `avoid` entries and the normal-difficulty Den rule (fewer than 2 healing potions) into `Exclude`, not only
  cleared areas.
- When the worthy area is avoided rather than cleared, `Pick` returns false. With `l.cleared` empty, `noTarget`
  (`goal.go:512-519`) calls `setError("... no out-of-game flow is wired")` even when a Cycler is wired, and waits
  `ErrorRetrySec` (300 s). In the campaign path `freshGameToLevel` returns not-done, so the step fails with KindNotFound,
  which feeds H3. S: it ends when the avoid expires, but it costs 15 min per occurrence and the log points the operator the
  wrong way.
- Fix: apply the floor only to areas in `l.cleared`, and in the no-cleared case fall back to the old "last eligible" pick.
  Make the message say why (avoided / floor).

### M5. Flee has no exit condition
- V `internal/resource/abort.go` (+`combat/resource.go:205-222`): with no potion, no TP and life under FleeLifePct (50 %),
  every fight tick moves 3x KiteRetreat away, with no limit.
- S: base life regeneration is slow, so a poor character in the field stops fighting for the rest of the Clear and kites
  until the deadline or the chicken line. The chicken exit then feeds H1. Check that the heal-first gate walks the
  character home before the next Clear.
- Fix: bound the number of consecutive flee ticks, then go home on foot (the walk-home path exists for the Den) or chicken
  deliberately. U: confirm `KitTPGold` 100 and `KitPotionGold` 30 against 1.14d vendor prices.

## Low
- L1 V data race: `launch.Recycler` fields `cycles`, `last` and `count` (`recycle.go:66-75`) are written on the runner
  goroutine (Recycle, NoteCycle) and read from HTTP (`rc.Status()` in `cmd/d2bot/live.go` State, and metrics) without a
  lock. Add a mutex. Also run `-race` on the worker host for internal/bot and cmd/d2bot.
- L2 S nested ladder: in rungs 2-4, `r.do` can end with cause errStuck, which calls `escalate` again from inside `escalate`
  (`loop.go:338`). The outer call then sets `stuckStage` back. Bounded, but it can run two rungs for one stall and repeat
  rung 3. Treat errStuck inside a rung as "rung failed" and do not recurse.
- L3 S `starveArea` without a TP (`leveling.go:173-182`) picks the first plan step of the difficulty, which is Act 1,
  even when the character is in a later act's town. The walk fails and is only rescued by skipTarget's avoid.
- L4 S `expectGameExit(false)` drops one health event. If a relaunched game dies inside the bracket after `Restart`
  returned nil, the monitor's exit is swallowed. The next snapshot error has no fallback because `d.Health != nil`
  disables the unreadable-memory check (`observe.go:116`). Harmless while the monitor keeps reporting (each new pid gives
  Up and then Exited), but there is no test.

## Recommended order before an unattended 24 h soak
1. H2 (menu-grace exemption for OOG actions) and H1 (narrow `real`, count gameLost in the re-entry breaker, add an hourly
   cap). These two decide whether the bot recovers or spins.
2. H3 (reset or decay executor counters per new game).
3. M1 (one retry and a single restart timeout), M3 (PendingGod check at start), M2 (stale-marker check).
4. A live soak on one instance, checked against `docs/stallfatal.md:41`: rung 4 completes and the goal continues, and a forced
   game kill restarts once and is counted.

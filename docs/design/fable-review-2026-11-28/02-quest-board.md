# Fable review 2 of 8: quest board, step executor, hand-in (changes of 2026-11-27)

> status: review, read-only. Base: origin/main at 415b11f25. No code was changed. Tags: V = read in the repo or run (said which), S = inferred, U = not verified.

Scope: `git log origin/main --since 2026-11-27` for internal/questboard, internal/bot/questboard.go, internal/bot/stepexec.go, internal/task, internal/questspec, data/questboard (cards akarafix, racefix, questresume, refusalpark, cellar3gate, questautovis, bsmprep, bsmchk, act2chunks, tyrael, denreward, act2handin) plus the difficulty go file the brief names.

What was run: `go test -count=1 ./internal/questboard/ ./internal/questspec/ ./internal/diffready/` on the worker host, once plain (green) and once with `QUESTBOARD_SM=1` (RED, finding 10). Everything else is by reading; no finding below was reproduced with a new test (no Go changes allowed), so each "must-fail test" is a sketch, not a result.

## Verdict on the quests.boardSM default flip (TASK-983)

Not tonight. A 2 h sandbox window through the flip tool (which restarts the child, so finding 5 does not bite) is fine on ONE sandbox, with `quests.resumeAfterDeaths` and `quests.refusalPark` left at 0. The code default should not flip until:

1. `QUESTBOARD_SM=1 go test ./internal/questboard` is green again (finding 10; it is the documented SM suite and it is red on main).
2. `board.park.persist-restart` has run green for two builds. docs/questboard.md ("boardSM legacy removal", gap list) says itself: the park drill has never run, `board.hold.timeout` was red, the Sorceress never ran restart-mid-chain (V, read).
3. A drill that restarts between the first and the second death on one chunk shows the chunk still parks (finding 1).
4. A flip, death, auto-revert, re-flip sequence is run once on a sandbox and the chunk that killed the hero does not start again after the revert (finding 1).
5. One live hand-in under boardSM 1 answers `done`, not `confirming` or `blocked` (racefix is "not verified live" on its own card; finding 8).
6. `data/darklaunch.json` row `quests.boardSM` still says `criteria_met: U`.

## Findings

### 1. HIGH (blocks the default flip): what leads to a park does not survive a restart, and a revert lifts every park
- V: the death and failure counters live in `runState` (internal/questboard/exec.go:101-116, "never persisted"); `Report` parks at `rs.deaths >= MaxDeaths` (exec.go:791-801) and only then writes `Run.Deaths` (exec_sm.go:239-262). A new `Exec` is built on every process start and on every hero switch (internal/bot/questboard.go:144-160), with an empty map.
- V: with boardSM 0 `smPark` returns at once (exec_sm.go:240) and the overlay that shows a stored park is not loaded (exec.go:181-188), so the legacy board never honours a park stored by the machine.
- Scenario A: boardSM 1, the hero dies once on Blood Raven, the child restarts (self-update, crash, flip window opening or closing), dies once more, restarts. `deaths` never reaches 2; the hero dies on that chunk once per process lifetime, for ever.
- Scenario B (two deciders undoing each other): boardSM 1 parks the chunk `died_too_often`. Any hard breach in the window (docs/process/dark-launch.md names a death among them; the watcher's own row list does not, U) restores the backup and restarts the child at boardSM 0. The legacy board sees an accepted record and no park, and runs the chunk again at once. The safety revert removes the safety stop.
- Recommendation: persist `Failures` and `Deaths` on the live run at every count, not only at the park; make the legacy `derive` treat a stored `RunParked` as blocked (read-only) so a revert keeps parks.
- Must-fail test: rig with SM; Report StepDied once; `r.restart()`; Report StepDied once; want `Next().Kind == DecBlocked`. Second: park with SM, `restart()` with `sm: false`, want `DecBlocked`, get `DecRun`.

### 2. HIGH under quests.refusalPark: the rung 4 skip treats every failed quest step as a refusal
- V: `noteStepRefused` counts any result with an error and outcome Retry, Blocked or Skipped (internal/bot/task.go:305-310), so a walk that failed, a dialogue that did not open and a path error all raise `refStall`. `refusalOnlyStall(3)` is `refStall >= 3` since the last kill, xp or area change (internal/bot/invariants.go:381, 589-593). It does not check that the stall consists only of refusals, and with boardSM 1 real guard refusals are holds that the board never reports as failures (internal/task/gate.go:317-324).
- Scenario: the game stops taking input (the case rung 4 exists for). Three step attempts fail with errors, the ladder reaches rung 4, the skip fires (internal/bot/loop.go:766-771), the bot pauses itself, the pause resets `refStall` (invariants.go:381), the automatic resume climbs again, three failures, skip again. The game process is never restarted. The knob that was meant to stop useless restarts hides the one failure a restart fixes.
- Recommendation: count only results with `Hold` or a code of kind hold toward `refStall`, and require that no uncoded failure happened in the stall.
- Must-fail test: invMon fed three `noteRefusal` calls from results with `Kind` nav error and no code; want `refusalOnlyStall(3) == false`.

### 3. HIGH under quests.resumeAfterDeaths: the park can never lift unless the goal is a campaign, and the page says it will
- V: `ResumeTick` is called only from `boardStep` (internal/bot/questboard.go:787-792). Under a ReachLevel, FarmArea or KillBoss goal `boardStep` runs only when `boardPreempts` is true, and for a parked head `boardPreemptsWhy` returns false (questboard.go:1292-1297, internal/bot/goal.go:103-105). Idle and Runs goals never reach it either (goal.go:52-57, 96-101); under a campaign it is skipped while the gold farm order, the purse recovery or the arbiter hold runs (goal.go:110-118).
- V: `ResumeNote` computes its sentence without any of that (internal/questboard/resume.go:139-149), so /api/quests shows "Resumes by itself in about N min" for a park that nothing will lift. The 42 h idle hero the card was written for stays parked if its goal is a level goal.
- Recommendation: call `ResumeTick` from the same place as `noteHandInHold` (goal.go:51), before the goal branches.
- Must-fail test: copy of TestQuestBoardStartRunsUnderAFarmGoal with a stored `died_too_often` park, fake clock advanced 46 min, life 100; want the run back to running.
- Bound on repeated deaths (the brief asks): V, it holds while the knob path is the only writer. `Retry` zeroes `AutoResumes` and `ResumeTick` writes `prior+1` (sm.go:440-441, resume.go:196-203), a second park is final (resume.go:104-106), so at most 4 deaths per chunk and 3 resumes a day (data/questboard/resume.json), counted from the plan file's event log (capped at 100 events, enough). It is broken by finding 1 (counter reset) and finding 7 (silent Reset).

### 4. HIGH under bot.diffReadyV2: the owner's go is consumed before the hop is possible, and its file is deleted
- V: `ensureDifficulty` calls `wantedDifficulty` first (internal/bot/goal.go:1225-1227); `wantedDifficultyV2` calls `diffGo.Take` whenever the hero is ready (internal/bot/diffready.go:167-169); `Take` deletes the go and rewrites or removes diffgo.json (internal/diffready/consent.go:88-98). Only afterwards come the backoff check, the "not in town" return for a level goal (goal.go:1233-1241) and the new game that can fail (goal.go:1250-1259).
- Scenario: the owner taps go while the hero levels in the field. The next tick takes the go, logs "difficulty wish not met ... inTown false" and returns; the following tick has no go. The hop never happens and the Return screen no longer shows it pending. A restart does not help: the file is gone. A failed new game loses it the same way.
- Second consumer: `DifficultyGate` (diffready.go:88-108, the chain in cmd/d2bot) also takes the go; whichever asks first wins (S: not traced which one acts on it).
- Recommendation: `Granted` to decide, `Take` only after `verifyDifficulty` succeeded.
- Must-fail test: runner in the field, ready, go granted; one `ensureDifficulty`; want `diffGo.Granted("nightmare")` still true.

### 5. MEDIUM: boardSM is read once by the executor and live by everything else
- V: `Options.SM` is set in `newQuestBoard` only (internal/bot/questboard.go:96); `boardRTFor` refreshes Enabled, Auto, Resume and Refusal but not SM (questboard.go:163-168); there is no setter. `World.BoardSM` follows the live knob every tick (internal/bot/task.go:372), and so does `kitPlanOn` (internal/bot/routeguard.go:441). The tuner writes the live tuning in place (internal/bot/tuner.go:125-132).
- Scenario: boardSM set to 1 from the tuning page (no restart). Guards now return coded holds, the executor still runs legacy: `holdLocked` reaches `hold_timeout`, `smPark` is a no-op, the park exists in memory only. The other direction after a page revert: the machine keeps running while guards return uncoded danger faults, which count as failures again.
- Recommendation: add `SetSM` next to `SetRefusal`, or mark the knob restart-only in its description and refuse a live edit.
- Must-fail test: runner test that edits the knob through the Tuner and wants `exec.SMOn()` to follow.

### 6. MEDIUM: cellar3gate took the no_path park out of the one retry a new release gives
- V: `smReleaseRetry` retries only `Code == "gave_up"` (internal/questboard/exec_sm.go:279-283). Since cellar3gate a navigation failure parks as `no_path` (exec.go:807-814), and since refusalpark a coded one as `refused_repeatedly` (exec.go:782-785). Both were `gave_up` before today.
- Scenario: the cellar mouth gate bug parks the Countess chunk as `no_path`. The nav fix ships. The park stays until the player taps Try again; before today the new release retried it once by itself.
- Recommendation: retry `gave_up`, `no_path` and `refused_repeatedly`.
- Must-fail test: copy of the release-retry test with code `no_path`.
- Also V: `parkCode` matches the text "nav: no path" and "cause=" of the LAST failure only (exec.go:810); a rewording in nav turns the park back into gave_up with no test noticing (S).

### 7. MEDIUM: switching quests.board off clears every stored park
- V: `SetEnabled(false)` calls `Reset` (exec.go:150-158); `Reset` calls `smRetryAll` (exec.go:609-611, exec_sm.go:347-357), whose comment says the player's Reset is the one explicit way a park is cleared. `boardRTFor` calls `SetEnabled` with the live knob on every use (questboard.go:164). Every park but game_failed is lifted and `AutoResumes` zeroed, with no player action: a flip window on `quests.board`, or any bundle that contains it, is a silent Try again on all parks, and it resets the "once in a row" bound of finding 3.
- Recommendation: `SetEnabled(false)` drops the in-memory state only.
- Must-fail test: stored park; SetEnabled(false); SetEnabled(true); want DecBlocked.

### 8. MEDIUM: hand-in mailbox, what can still go wrong after akarafix and racefix
- V, tapper gone in mid-serve: the ctx is checked at dequeue only (questboard.go:1070); the dialogue then runs up to 3 min. If the tapper's ctx ends meanwhile, `Exec.HandIn` returns the ctx error and clears `handing` (exec.go:894-900), the dialogue still completes and sets `handed` (questboard.go:1103). The run closes by itself at the next derive (latch), but the awards are settled only inside `HandIn` (exec.go:923), so they wait for another tap; a second tap in that window runs the dialogue a second time. Low impact, real.
- V, failed attempts are never re-read: on a dialogue error the loop `continue`s without a derive (exec.go:906-910). For an act hop `handed` is forced false (questboard.go:559-563), so the answer depends on the rebuilt world. `worldLoop` reads `r.latest()` and returns early without touching the cache when there is no in-game snapshot (questboard.go:239-242), which is the state right after a boat ride (S, U live). Then attempt 1 reads "did not register", attempts 2 and 3 fail on the "go east" line that Act 2 does not offer, and the reply is `blocked: hand-in failed 3 times` for a hop that worked; the park is cleaned by the next derive. Recommendation: derive before every attempt and return Done when the chunk already reads done; in `boardHandIn` force one observation (`r.poll()`) before `worldLoop`.
- V, not served: `boardServeHandIn` has three call sites (goal.go:54, 97; questboard.go:794). A campaign hero in gold farm, purse recovery or under the arbiter hold passes none of them, in town or not (goal.go:110-118); the tap times out with a reason. Acceptable if intended; say so in docs/questboard.md.
- V, the replace-a-dead-request branch of `enqueue` (questboard.go:658-668) is unreachable in practice: `Exec.handing` admits one `HandIn` per hero (exec.go:883-886) and a leaving tapper removes its own request (`dropOwn`). The capacity-1 races the brief asks about (a live request lost when put back into a refilled slot) need two concurrent enqueues, which that flag forbids. No finding, but the branch has no caller that can reach it; keep the test, do not build on it.
- Lock order (asked): V by reading, no cycle found. Order is `Exec.mu` then (try) `stepMu` then `boardRT.mu`; the loop holds `stepMu` and takes `Exec.mu`, API goroutines hold `Exec.mu` and only TryLock `stepMu`; nothing takes `Exec.mu` while holding `boardRT.mu`. One consequence: inside a long step (a 40 min slice, questboard.go:1007) every API read and `boardYield` sees the world of the step's start.

### 9. LOW: denreward holds trips once per pending stretch, not once per reward
- V: `rewardSince` is cleared only when no quest at all reads cleared (goal.go:384-396); `questRewardPending` is true for any quest at StageCleared (goal.go:399-414). With one hand-in left untapped (a chunk awaiting the player for hours) the 2 min bound is spent, and the next quest cleared in the field gets no hold; the Den loss the card fixed (portal trip right after the clear) is back for that quest. Recommendation: key the hold by quest id.
- Must-fail test: `holdTripForReward` driven with pending quests {A}, 3 min later {A, B}; want a hold for B.

### 10. MEDIUM as evidence: the documented SM suite is red on main
- V (run on the worker host): `QUESTBOARD_SM=1 go test -count=1 ./internal/questboard/` fails `TestBoardLegacySwitchIdle` (legacy_test.go:22, "boardLegacy() = false with SM false"); plain run green. The seam test of bsmprep builds its rig by hand and the env switch overrides `sm: false`. A test artifact, but docs/questboard.md names exactly this command as the SM suite, so the suite was not run after bsmprep merged.
- Recommendation: make the rig's `sm` field win over the env in that test; add the env run to tools/ci/check.sh until the flip.

### Smaller notes
- V: `execIndex` (internal/bot/stepexec.go) files talk, interact, kill and transmute rows in maps with no duplicate check; a second row with the same NPC and line, object or monster silently replaces the first. A scan of data/questspec/act1.json and act2.json found no collision today (V, script). Recommendation: a questspec rule 15 that refuses duplicate adapter keys.
- V: the events EvHandInTap, EvHandInDone, EvHandInFail, EvObjective, EvRefuse, EvClear and EvRestart of sm.go are applied nowhere outside the table; stored runs are only queued, running, parked, finished or stopped. The design's "restart while held or awaiting" gap therefore has nothing stored to lose; the dead table cells should be deleted or wired with the legacy removal.
- S: the refused_repeatedly park is close to unreachable with boardSM 1. Guard refusals are holds; the only coded non-hold fault is needs_potions after its chores insert was given up (internal/task/gate.go:306-309, 351-357, 364-367). The knob's visible effect is mostly finding 2.
- V: `ResumeInput.LastDeath` is per process (internal/bot/questauto.go:60-70); after a restart the quiet stretch runs from `ParkedAt` alone. Safe direction only if deaths before the restart are older than the park, which they are.

## Dead ends
- Double count of one death by `boardRun`: no. The loop returns to `deathWait` before any goal step while the hero is dead (internal/bot/loop.go:95).
- Self-deadlock of `world()` on `boardRT.mu` through `taskEnv.World`: no path found that re-enters the executor.
- Day cap evasion through log trimming: no, 100 events against caps of 3 and 6 a day.

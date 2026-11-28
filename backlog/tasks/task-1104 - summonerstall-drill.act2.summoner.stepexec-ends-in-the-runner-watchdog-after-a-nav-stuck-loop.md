---
id: TASK-1104
title: 'summonerstall: drill.act2.summoner.stepexec ends in the runner watchdog after a nav stuck loop'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:19:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 528000
card: "summonerstall"
lane: next
owns:
  - "internal/bot/"
  - "internal/nav/"
  - "cmd/d2bot/boss.go"
  - "cmd/d2bot/boss_test.go"
  - "backlog/tasks/task-1107 - act2night-three-Act-2-stepexec-drill-findings-Duriel-stall-shaft-road-without-a-portal-potion-not-drunk-in-the-cube-drill.md"
exclusive: []
contracts: []
result: "Cause found offline (not verified live): in the Arcane Sanctuary the Summoner at 26,450 is unreachable on the nav map; each fight approach re-ran the full 900x900 search plus 24 teleport pad searches (about 0.5 s wall, almost no game time), so the boss attack slice (wall clock, gclock) ran out first, cmd/d2bot mapKillErr read that as no error, the boss fail counter never grew and the runner watchdog fired at 614 s wall with 36 s of game time. Fix: internal/nav asks a failed plan once per field (nopath.go, test TestUnreachableGoalIsSearchedOnce), so the fight fails fast and the boss runner ends after 5 failed attacks. Proposed for cmd/d2bot (not owned): mapKillErr should return the fight error even when the slice also ended. The reachability of the Summoner itself (teleport pads none usable) stays open (class D). nf3b (not verified live): the Summoner was reachable; the hero walked onto his platform twice (48,455 and 33,472), the stalls of those walks hard-blocked cells on the way (102,503 / 83,462 / 48,455 / 50,458, game-time TTL 180 s, 36 s passed) and the plan retry without hard blocks never tried the seams the Arcane Sanctuary grid needs, so the third walk found no way (U that those cells lie on a seam route: the level grid was not kept). Fix: planSearch tries seams also without the hard blocks (internal/nav/nav.go, test TestPlanCrossesASeamBehindAHardBlock, class-independent, run as Paladin and Sorceress). Not the pads (the pad arm is another arm; the one pad tried was 368 subtiles away), not the goal cell. cmd/d2bot mapKillErr now returns a fight's own error when the slice also ended (boss.go, test TestMapKillErrAndBossErr). Nothing build-specific. Open: hopPad marks a pad used before the jump is seen."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Run 20261127-231648: BLOCKED after 647 s wall. Last lines: 'nav: stuck recovered method=replan rung=3 repeat=true', then ERROR 'watchdog: runner loop made no progress, exiting for a restart age=614s limit=600s' with a goroutine dump (bridgeLoop, watch.Hub.Serve). U: the watchdog age is wall clock, so a slow lockstep slot at load 7 may count as a stall; the dump decides. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The stalled goroutine is named from the watchdog dump in the run log
- [ ] #2 drill.act2.summoner.stepexec Paladin 11 is judged (GREEN or RED), not BLOCKED
<!-- AC:END -->

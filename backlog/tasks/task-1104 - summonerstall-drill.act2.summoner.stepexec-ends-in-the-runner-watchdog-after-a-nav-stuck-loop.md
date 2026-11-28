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
exclusive: []
contracts: []
result: "Cause found offline (not verified live): in the Arcane Sanctuary the Summoner at 26,450 is unreachable on the nav map; each fight approach re-ran the full 900x900 search plus 24 teleport pad searches (about 0.5 s wall, almost no game time), so the boss attack slice (wall clock, gclock) ran out first, cmd/d2bot mapKillErr read that as no error, the boss fail counter never grew and the runner watchdog fired at 614 s wall with 36 s of game time. Fix: internal/nav asks a failed plan once per field (nopath.go, test TestUnreachableGoalIsSearchedOnce), so the fight fails fast and the boss runner ends after 5 failed attacks. Proposed for cmd/d2bot (not owned): mapKillErr should return the fight error even when the slice also ended. The reachability of the Summoner itself (teleport pads none usable) stays open (class D)."
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

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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Run 20261127-231648: BLOCKED after 647 s wall. Last lines: 'nav: stuck recovered method=replan rung=3 repeat=true', then ERROR 'watchdog: runner loop made no progress, exiting for a restart age=614s limit=600s' with a goroutine dump (bridgeLoop, watch.Hub.Serve). U: the watchdog age is wall clock, so a slow lockstep slot at load 7 may count as a stall; the dump decides. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The stalled goroutine is named from the watchdog dump in the run log
- [ ] #2 drill.act2.summoner.stepexec Paladin 11 is judged (GREEN or RED), not BLOCKED
<!-- AC:END -->

---
id: TASK-1035
title: 'faultstall: the bot keeps making progress under the fault proxy'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T06:21:00Z'
updated_date: '2026-11-27T07:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 500000
card: "faultstall"
lane: next
owns:
  - "internal/faultproxy/"
  - "cmd/d2scenario/faults.json"
  - "docs/probes/faultsoak-design-2026-11-26.md"
exclusive: []
contracts: []
result: "Cause: the fault sets counted commands, so a 20 s held reply came every few frames and the bot waited most of the time, one runner step outlived the 10 minute watchdog; the proxy also kept the game slot and its lock during a hold after the bot hung up, so redials met err busy. Fix: rates in time, a hold ends when the bot hangs up, unknown set keys rejected. Verified offline with a single-client fake game, not live. Not build-specific. closecards3 Remaining (live): #4 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: every feature-soak fault row stopped on the d2bot watchdog after about 10 minutes with no minute line (docs/probes/feature-soak-2026-11-26.md 3.4). Done when: the cause is found from the row logs, reproduced offline and fixed with a must-fail test.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause of the 10 minute watchdog stops under the fault sets found from the row logs
- [x] #2 Must-fail offline test reproduces it with a fake single-client d2d
- [x] #3 Fix in the proxy or the bot, tests pass
- [ ] #4 Live soak row under the mixed set logs minute lines (live check)
<!-- AC:END -->

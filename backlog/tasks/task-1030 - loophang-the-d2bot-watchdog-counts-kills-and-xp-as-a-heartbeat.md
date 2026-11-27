---
id: TASK-1030
title: 'loophang: the d2bot watchdog counts kills and xp as a heartbeat'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T02:14:00Z'
updated_date: '2026-11-27T03:34:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 498000
card: "loophang"
lane: next
owns:
  - "internal/bot/observe.go"
  - "internal/bot/runner.go"
  - "internal/bot/runner_test.go"
exclusive: []
contracts: []
result: "LastStep now also counts the last kill, xp gain or level (atomic, no lock). Offline test only, not verified live. Quit on clean start does save (d2d log), the xp reset after the restart is unexplained."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: Doofer on d2play-20261108 was restarted by the watchdog after 613 s inside one Den of Evil Clear action although it killed 16 monsters in that time. Done when: kills, xp and levels count as a heartbeat and a hung step is still caught.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A long action that keeps killing no longer trips the 10 minute watchdog (test fails without the fix)
- [x] #2 A hung step with no kill, xp or finished step still trips it
<!-- AC:END -->

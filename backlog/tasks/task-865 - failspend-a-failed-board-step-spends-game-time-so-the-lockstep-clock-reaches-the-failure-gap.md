---
id: TASK-865
title: >-
  failspend: a failed board step spends game time so the lockstep clock reaches
  the failure gap
status: Done
assignee:
  - credit-runner
created_date: '2026-11-23T06:51:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 391000
card: "failspend"
owns:
  - "internal/bot/questboard.go"
  - "internal/bot/questboard_test.go"
exclusive: []
contracts: []
result: "Cause of the 49-minute freeze (necro.act1-stage1, Necromancer 11): a board step refused with a ui error (no exit) ran nothing, the tick loop returned true without any clock sleep, so under lockstep no game time passed, the 15 s failure gap never elapsed and the 3 counted failures that park a chunk never came; the loop polled snapshots 130 times a second. Fix: spend boardFailGap/3 after a failed non-idle step. A lab-visible bug (real play has a wall clock) but also a tight CPU spin there. Offline unit test only; rerun of the staged drill after the merge."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A failed non-idle board step sleeps a third of boardFailGap
- [x] #2 Unit test for failSpend
<!-- AC:END -->

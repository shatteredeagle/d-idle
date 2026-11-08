---
id: TASK-165
title: 'flakepanicsafe: deflake panic-safe runner test and drought info event'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-08T03:21:00Z'
labels:
  - bot
  - bug
  - gate
milestone: m-4
dependencies: []
priority: high
ordinal: 131000
card: "flakepanicsafe"
owns:
  - "internal/bot/panicsafe_test.go"
  - "internal/bot/loop.go"
  - "internal/bot/goal.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Flake on main CI run 42440804633. Done when the wait is info and tests are stable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fresh-game wait emits an info event, not an error
- [x] #2 Regression test fails before the fix
- [x] #3 TestPanicIsRecovered passes 50x under load
<!-- AC:END -->

result: Root cause: the fresh-game wait was an EvError, racing the test error-event scan (V: regression test fails before). 1000 runs (20 parallel x50, -cpu 1) pass. Not verified live.

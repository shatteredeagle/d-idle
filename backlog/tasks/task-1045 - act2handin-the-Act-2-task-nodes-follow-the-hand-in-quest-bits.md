---
id: TASK-1045
title: 'act2handin: the Act 2 task nodes follow the hand-in quest bits'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T13:22:00Z'
updated_date: '2026-11-27T13:49:00Z'
labels:
  - bot
  - bug
dependencies:
  - TASK-1044
priority: medium
ordinal: 507000
card: "act2handin"
lane: next
owns:
  - "internal/task/act2.go"
  - "internal/task/act2handin.go"
  - "internal/task/act2handin_test.go"
exclusive: []
contracts: []
result: "The Duriel and Act 2 hand-in nodes now read the Seven Tombs quest bits from the questspec rows: Tyrael is done on his bits, Jerhyn runs after Tyrael and before Meshif, Meshif waits for the Jerhyn bit; without a raw word the old stage checks apply; unit tests on two builds, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tyrael's node is done on his bits, not on the stage
- [x] #2 Jerhyn's node runs after Tyrael and before Meshif, whatever the Act 2 word says
- [x] #3 Must-fail tests replay 0x0004, 0x0024, +0x2000, +0x0010, 0x0001 on two builds
- [ ] #4 Live: a drill hands in kill, Tyrael, Jerhyn, Meshif and ends with bit 0x0001
<!-- AC:END -->

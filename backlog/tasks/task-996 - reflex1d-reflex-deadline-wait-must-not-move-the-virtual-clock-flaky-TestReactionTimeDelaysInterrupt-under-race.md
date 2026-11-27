---
id: TASK-996
title: >-
  reflex1d: reflex deadline wait must not move the virtual clock (flaky
  TestReactionTimeDelaysInterrupt under -race)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T05:50:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 481000
card: "reflex1d"
lane: next
owns:
  - "internal/bot/safetylat.go"
  - "internal/bot/loop.go"
  - "internal/bot/reflexdrink_test.go"
exclusive: []
result: "holdForHard waits on a gclock timer (done or the reflex deadline) instead of Clock.Sleep, so it no longer advances a virtual clock; race-checked on the worker host."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Gate failure on #1341. Done when -race -count=5 is green on the worker host.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 the guard's wait after a hard cancel uses a timer, not Clock.Sleep; TestReactionTimeDelaysInterrupt passes -race -count=5
<!-- AC:END -->

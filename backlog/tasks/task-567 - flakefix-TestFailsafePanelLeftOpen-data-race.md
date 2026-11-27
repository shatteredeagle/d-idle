---
id: TASK-567
title: 'flakefix: TestFailsafePanelLeftOpen data race'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T21:52:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 296000
card: "flakefix"
owns:
  - "internal/bot/failsafe_test.go"
exclusive: []
contracts: []
result: "Root cause: the test assigned g.r.d.Loot after g.run() started the runner goroutine, which reads d (data race under -race, not timing). Moved the assignment before run. Not live (offline test). closecards3: ticked #2 (go test -race -count=500 -run TestFailsafePanelLeftOpen ./internal/bot ok on the worker host 2026-11-27 (V))."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Race fixed (Loot set before the runner starts)
- [x] #2 go test -race -count=500 green
<!-- AC:END -->

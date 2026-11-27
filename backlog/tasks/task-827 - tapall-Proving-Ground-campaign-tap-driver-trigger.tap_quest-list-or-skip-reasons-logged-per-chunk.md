---
id: TASK-827
title: >-
  tapall: Proving Ground campaign tap driver (trigger.tap_quest list or *), skip
  reasons logged per chunk
status: Done
assignee:
  - credit-runner
created_date: '2026-11-23T00:14:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 378000
card: "tapall"
owns:
  - "cmd/d2scenario/tap.go"
  - "cmd/d2scenario/tap_test.go"
  - "internal/scenario/scenario.go"
exclusive: []
contracts: []
result: "tapper: trigger.tap_quest as a comma list or *, per-chunk why-not log; offline tests only. Resume from a previous stage save is NOT supported by the runner (every run creates its own hero named by scenario id), so stages start from declared level fixtures."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tap_quest accepts a list or *; taps the row's own enabled action in board order
- [x] #2 Reason for each skipped chunk logged once per change; must-fail test
<!-- AC:END -->

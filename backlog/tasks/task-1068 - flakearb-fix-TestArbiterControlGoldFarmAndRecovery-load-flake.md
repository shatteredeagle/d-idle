---
id: TASK-1068
title: 'flakearb: fix TestArbiterControlGoldFarmAndRecovery load flake'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T21:33:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 514000
card: "flakearb"
lane: next
result: "Cause: the guard goroutine raced the fake walk from area 3 to the farm zone 4 and cancelled the slice (crossed an area border); test now starts in the zone. Not verified live (offline test). Before ~5% per run under load, after 0 of 300."
owns:
  - "internal/bot/arbiter_control_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Root cause found and fixed in the test
- [x] #2 100x3 cpus under load passes
<!-- AC:END -->

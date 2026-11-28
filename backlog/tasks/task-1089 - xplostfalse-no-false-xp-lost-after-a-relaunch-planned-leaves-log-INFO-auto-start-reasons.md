---
id: TASK-1089
title: >-
  xplostfalse: no false xp lost after a relaunch, planned leaves log INFO,
  auto-start reasons
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T02:39:00Z'
updated_date: '2026-11-28T03:24:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 522000
card: "xplostfalse"
lane: next
result: "xp fall confirmed by a second read (two-class test); planned Save and Exit logs INFO with relaunch reason; auto-start held-back/cancelled reasons log once and show in ops status --why. Offline only, not verified live."
owns:
  - "internal/questboard/auto.go"
  - "internal/questboard/auto_test.go"
  - "internal/bot/observe.go"
  - "internal/bot/xploss_test.go"
  - "internal/bot/fixrestart_test.go"
  - "internal/bot/relaunch.go"
  - "internal/d2denv"
  - "cmd/d2bot/d2dbackend.go"
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
- [x] #1 xp fall needs a second consistent read
- [x] #2 planned Save and Exit logs INFO with reason
- [x] #3 auto-start held-back reasons logged once and in ops status --why
<!-- AC:END -->

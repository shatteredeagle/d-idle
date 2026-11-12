---
id: TASK-358
title: >-
  lockfix: roam done() and NotePosition no longer nest covMu and obMu (sandbox
  deadlock)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-12T04:29:00Z'
updated_date: '2026-11-12T04:45:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 233000
card: "lockfix"
owns:
  - "internal/nav/roam.go"
  - "internal/nav/roam_test.go"
  - "internal/bot/observe.go"
exclusive: []
contracts: []
result: "roam() no longer calls done() under covMu (V: regression test fails on old code); observe defers NotePosition until obMu is released, so the two locks never nest. Introduced-before: the done()-under-covMu call pre-existed (hostile()->latest() takes obMu), exploredEnough widened it. Race and tests on the worker host; not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 done() never called with covMu held (regression test)
- [x] #2 NotePosition runs after obMu is released
- [x] #3 gates green
<!-- AC:END -->

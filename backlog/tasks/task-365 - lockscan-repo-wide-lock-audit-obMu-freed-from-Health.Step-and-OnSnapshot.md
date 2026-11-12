---
id: TASK-365
title: 'lockscan: repo-wide lock audit, obMu freed from Health.Step and OnSnapshot'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T16:21:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 237000
card: "lockscan"
owns:
  - "docs/locks.md"
  - "internal/bot/observe.go"
  - "internal/bot/runner.go"
  - "internal/bot/lockorder_test.go"
  - "internal/bot/lockguard_test.go"
exclusive: []
contracts: []
result: "Audit of ~190 mutex sites (AST scan of lock regions for callbacks, plus manual read of the paths in the incident dump). 3 real hits fixed, all under Runner.obMu: Deps.OnSnapshot (display tracker) and launch Health.Step (d2d control-socket ping) in pollWith, and Health.Step in expectGameExit. Regression tests with a 2 s deadlock detector fail on the old code; AST guard test TestNoCallbackUnderObMuOrCovMu; docs/locks.md has the lock order. Suspected, unproven: Quests.Read under obMu (d2d QuestSource, 5 s bound). Verified offline only, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the 2026-11-12 sandbox freeze (lock inversion obMu/covMu, TASK-358) needs a repo-wide audit for the same class. Done when: hits fixed with tests, docs/locks.md written, guard test in place.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 audit done, hits fixed with regression tests
- [x] #2 docs/locks.md written
- [x] #3 guard test added
<!-- AC:END -->

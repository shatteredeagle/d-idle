---
id: TASK-242
title: 'areathrash: escalating border-loop breaker, cave ping-pong analysis'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T01:41:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 180000
card: "areathrash"
owns:
  - "docs/areathrash.md"
  - "internal/bot/areaskip_test.go"
  - "internal/bot/loop.go"
  - "internal/bot/runner.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fun killer 5 of docs/research/beta-readiness-review.md (1314 zone entries in 11.7 h). Done when: the cause is documented and the breaker stops the cave shuttle early (docs/areathrash.md).

result: offline only, NOT verified live. Breaker fix unit tested (TestBorderLoopSkipEscalates); the arrival bounce itself is unexplained (U).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Log analysis of the sandbox day documented in docs/areathrash.md
- [x] #2 Border-loop breaker trips faster and its skip escalates per area, unit tested
- [x] #3 Dark Wood exit and cave-mouth hold causes explained (arrival bounce)
<!-- AC:END -->

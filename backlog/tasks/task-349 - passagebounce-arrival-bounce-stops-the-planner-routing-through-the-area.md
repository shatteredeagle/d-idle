---
id: TASK-349
title: 'passagebounce: arrival bounce stops the planner routing through the area'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T22:45:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 228000
card: "passagebounce"
owns:
  - "internal/bot/loop.go"
  - "internal/bot/lifelog.go"
  - "internal/bot/observe.go"
  - "internal/bot/runner.go"
  - "internal/bot/areaskip_test.go"
  - "docs/passagebounce.md"
exclusive: []
contracts: []
result: "Root cause (V, 3 h sandbox log): d2d fires the return exit of the Underground Passage and the Crypt about 0.2 s after the first move order whatever its direction (arrival bounce, 160 of 166 Passage arrivals); the old breaker only counted clearing-state crossings, travel walks never reached it. Fix: per-area bounce counter in observe (noteArrival/noteBounceLocked) trips the escalating border-loop skip, which bans routes through the area; arrival and bounce lines logged. Offline only, not verified live; why the game bounces stays U."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Root cause from the sandbox log is written down (docs/passagebounce.md)
- [x] #2 Arrival bounces are counted per area and trip the escalating border-loop skip in travel and clearing
- [x] #3 Arrival position and every bounce are logged
- [ ] #4 Whole-repo gate on the worker host passes
<!-- AC:END -->

---
id: TASK-351
title: 'arrivalsettle: the priming click no longer lands on the return warp'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T23:41:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 229000
card: "arrivalsettle"
owns:
  - "internal/d2dctl/verbs.go"
  - "internal/d2dinput/controller.go"
  - "internal/d2dinput/controller_test.go"
  - "internal/d2dinput/run_test.go"
  - "internal/bot/lifelog.go"
  - "internal/bot/observe.go"
  - "docs/passagebounce.md"
exclusive: []
contracts: []
result: "Cause (S from code, matches the V log pattern): d2dctl Kick clicked (400,300), the player feet, which on arrival is the return warp unit. Now clicks 110 px toward the move target (primeToward/KickAt). Offline, not verified live; measure bounces/arrivals from the arrival log line."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Priming click goes toward the move target, not onto the player's feet
- [x] #2 Arrival log line carries arrivals and bounces counters
- [ ] #3 Whole-repo gate passes
<!-- AC:END -->

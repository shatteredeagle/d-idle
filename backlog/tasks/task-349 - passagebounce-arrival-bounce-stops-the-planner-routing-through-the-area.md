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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Root cause from the sandbox log is written down (docs/passagebounce.md)
- [ ] #2 Arrival bounces are counted per area and trip the escalating border-loop skip in travel and clearing
- [ ] #3 Arrival position and every bounce are logged
- [ ] #4 Whole-repo gate on the worker host passes
<!-- AC:END -->

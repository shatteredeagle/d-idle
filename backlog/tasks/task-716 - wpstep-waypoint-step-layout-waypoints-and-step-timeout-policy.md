---
id: TASK-716
title: 'wpstep: waypoint step layout waypoints and step timeout policy'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:38:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 341000
card: "wpstep"
owns:
  - "internal/task/executor.go"
  - "internal/task/timeoutpolicy_test.go"
  - "internal/d2map/static.go"
  - "internal/d2map/tracker.go"
  - "internal/d2map/tracker_test.go"
  - "internal/d2dworld/"
  - "internal/bot/task.go"
  - "tools/provingground/scenarios/cain.json"
exclusive: []
contracts: []
result: "Partly done, NOT verified green. Layout objects of Kind waypoint now count as waypoint presets whatever their class (must-fail test, d2dworld); step timeouts extend once only while exploration progresses, then skip or park with the reason 'step timeout N (explored X%)' (task executor tests, Paladin and Sorceress). Scenario act1.wp.dark-wood(.layout) is RED on the builder (layoutRead 0 and 1) and marked pending: debug levelmap lists only the objects of rooms already built, so the Dark Wood waypoint is not named ahead (V builder jobs 20261120-234605-ae5b, 20261121-001400-6854). Next: the waypoint room's LvlPrest def. Not verified live in the sandbox. Remaining: AC 3 act1.wp.dark-wood green on the builder (RED at merge)."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the sandbox hero explored the Dark Wood for an hour on the Activate waypoint step. Done when: the Dark Wood waypoint scenario is green and a step cannot spin on timeouts.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 RED test: layout waypoint of unknown class is found
- [x] #2 step timeout policy: one extension on progress, then park with reason
- [ ] #3 scenario act1.wp.dark-wood green on the builder (layoutRead 0 and 1)
<!-- AC:END -->

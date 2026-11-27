---
id: TASK-716
title: 'wpstep: waypoint step layout waypoints and step timeout policy'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:38:00Z'
updated_date: '2026-11-27T20:02:00Z'
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
result: "Done on the builder, NOT verified live in the sandbox. Layout objects of Kind waypoint count as waypoint presets whatever their class (must-fail test, d2dworld); step timeouts extend once only while exploration progresses, then skip or park with the reason 'step timeout N (explored X%)' (task executor tests). The RED of act1.wp.dark-wood(.layout) (debug levelmap lists only built rooms) was fixed by TASK-770 levelobjects: fork levelmonsters objpresets (all rooms, built or not) merged by d2dworld.mergeObjectPresets; the scenarios are GREEN on the builder (Paladin 11, Sorceress 3, CHEATED setup, mechanics only); offline regression d2dworld TestObjectPresetsGiveTheWaypointAtArrival and TestLayoutReadWaypointKindCountsWithoutTableClass pass. Earlier RED reading (levelmap alone) is superseded."
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
- [x] #3 scenario act1.wp.dark-wood green on the builder (layoutRead 0 and 1)
<!-- AC:END -->

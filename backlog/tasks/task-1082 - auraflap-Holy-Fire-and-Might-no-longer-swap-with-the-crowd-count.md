---
id: TASK-1082
title: 'auraflap: Holy Fire and Might no longer swap with the crowd count'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T00:15:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 518000
card: "auraflap"
lane: next
owns:
  - "internal/combat/aura.go"
  - "internal/combat/fighter.go"
  - "internal/combat/auraflap_test.go"
  - "internal/combat/class_test.go"
  - "internal/combat/lifeprice_test.go"
  - "internal/agency/view.go"
  - "internal/agency/rotation_test.go"
  - "internal/agency/planner_test.go"
exclusive: []
contracts: []
result: "Root cause S: not two deciders (Drive answers empty at mode 1) but the profile ladder itself: Holy Fire needs a crowd count that moves with every kill and the no-target upkeep calls fell back to Might. Holy Fire is now held through the fight and 30 s after it (aura.go auraHold, no knob); the shadow agree flag includes the aura (the planner says Might, the profile Concentration or Holy Fire). Offline only, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Questy alternated Holy Fire and Might 95 and 97 times in 70 min.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 aura sequence over 200 ticks changes at most once per fight start and end (two builds)
- [x] #2 planner shadow agree flag includes the aura pick
<!-- AC:END -->

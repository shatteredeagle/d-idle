---
id: TASK-1088
title: 'holereach: zone picker drops unreachable areas, hole fallback needs the kit and logs once an hour'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T02:36:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 522000
card: "holereach"
lane: next
owns:
  - "internal/bot/zonemodel.go"
  - "internal/bot/zonemodel_test.go"
exclusive: []
contracts: []
result: "Zone model candidates are filtered by r.unreachable (cleared yardsticks kept); a hole fallback into another area needs fullKit and logs once per area per hour; offline tests only, not verified live. Knob-free."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Unreachable zones are not offered to the controller
- [x] #2 Hole fallback needs the plan kit and is logged once per area per hour
<!-- AC:END -->

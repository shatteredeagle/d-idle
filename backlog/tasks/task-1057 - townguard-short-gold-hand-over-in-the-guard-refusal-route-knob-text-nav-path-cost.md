---
id: TASK-1057
title: >-
  townguard: short-gold hand-over in the guard refusal, route knob text, nav
  path cost
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T19:22:00Z'
updated_date: '2026-11-27T20:06:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 511000
card: "townguard"
lane: next
owns:
  - "internal/task/"
  - "internal/bot/"
  - "internal/env/tuning_town.go"
  - "internal/town/route.go"
  - "internal/town/route_test.go"
  - "internal/nav/pathcost.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "offline only, not verified live: the guard needs_gold_farm hold says the supply Handover text (needs N more gold for X: sell, then recovery) via KitPlan.Hand, typed outcome and one log line per change unchanged; knob text and docs/knobs.md fixed; route.go prices legs with Navigator.PathCost (all legs or straight line), stop penalty is routeCosts.StopPenalty (U: value is a judgment). Build-neutral."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 KitPlan.Hand in the needs_gold_farm refusal, two builds
- [x] #2 knob text and docs/knobs.md regenerated
- [x] #3 route prices legs by nav path cost, stop penalty as data, order-change test
<!-- AC:END -->

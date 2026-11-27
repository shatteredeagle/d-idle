---
id: TASK-1039
title: 'upkeepcmp: shadow mode counts every comparison, not only disagreements'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T07:54:00Z'
updated_date: '2026-11-27T07:54:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 502000
card: "upkeepcmp"
result: "With combat.deliveryUpkeep at 1 every shadow comparison of the buff, aura, minion, mark and stance drivers is counted with its agreements, and the minute line carries upkeep=kind:agree/cmp while the knob is on. Offline tests only; not verified live."
lane: next
owns:
  - "internal/combat/upkeepcmp.go"
  - "internal/combat/upkeepcmp_test.go"
  - "internal/combat/aura.go"
  - "internal/combat/fighter.go"
  - "internal/combat/mark.go"
  - "internal/combat/minions.go"
  - "internal/combat/stance.go"
  - "internal/bot/upkeepkv.go"
  - "cmd/d2bot/adapters.go"
  - "internal/bot/observe.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Live: 0 upkeep shadow lines in 15 min on a Paladin could not tell agree from never compared. Done when every shadow comparison is counted on the minute line.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 mode 1 counts cmp and agree per driver kind (buff, aura, minion, mark, stance)
- [x] #2 minute line carries upkeep=kind:agree/cmp only when combat.deliveryUpkeep is non-zero
- [x] #3 test: Paladin aura up counts aura comparisons all agreeing; Sorceress counts none; mode 0 no field
- [ ] #4 live: sandbox Paladin minute line shows aura comparisons
<!-- AC:END -->

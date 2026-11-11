---
id: TASK-285
title: 'mercgate: merc equip and strip paths under gearDecisions=player, refuse a replacing hire while the merc wears gear'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T01:05:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 204000
card: "mercgate"
owns:
  - "internal/merc"
  - "cmd/d2bot/mercwire.go"
  - "cmd/d2bot/mercwire_test.go"
  - "docs/gear-policy.md"
exclusive: []
contracts: []
result: "Offline only, nothing live (2026-11-10): HireStep refuses a replacing hire while the merc wears gear or his gear is unread, under resource.gearDecisions=player; TestHireStepRefusesWhileMercWearsGear (Sorceress, Paladin; Rogue, Desert Guard). Legacy path kept under gearDecisions=1."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Stage A of docs/research/merc-gear.md (TASK-283). Done when the gate and its tests are merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A replacing hire is refused while the merc wears gear or his gear is unread (gearDecisions=player)
- [x] #2 Tests for Sorceress and Paladin, Rogue and Desert Guard
- [x] #3 gear-policy.md row updated
<!-- AC:END -->

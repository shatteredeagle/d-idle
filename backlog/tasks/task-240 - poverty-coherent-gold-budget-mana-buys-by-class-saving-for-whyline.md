---
id: TASK-240
title: 'poverty: coherent gold budget, mana buys by class, saving-for whyline'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T01:36:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 180000
card: "poverty"
owns:
  - "internal/resource/"
  - "internal/town/shop.go"
  - "internal/town/town.go"
  - "internal/bot/leveling.go"
  - "internal/bot/goal.go"
  - "internal/bot/whyline.go"
  - "internal/watch/whyline.json"
  - "internal/watch/whychange.json"
  - "internal/watch/whyline_test.go"
  - "docs/whyline.md"
exclusive: []
result: "Offline only, not verified live: entry band (empty belt), ManaBuy by ManaNeed, SavingFor, saving_for whyline, sell marks in sellable. Also edits internal/town/shop.go chores (mana purchase gate)."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fun killer 4 of the beta readiness review. Done when: see the criteria.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Poverty entry/exit band (no flip per potion)
- [x] #2 Mana potions by class mana use (Paladin and Sorceress tests)
- [x] #3 Whyline saving_for (watch template test); sell marks counted in sellable (UNTESTED, no rig)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

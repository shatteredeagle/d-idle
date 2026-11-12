---
id: TASK-328
title: 'potiondrink: bot tracks a running potion before it drinks again'
status: Done
assignee:
  - pro-runner
created_date: '2026-11-11T19:21:00Z'
updated_date: '2026-11-12T01:38:00Z'
labels:
  - bot
  - bot-fix
  - fidelity
dependencies: []
priority: medium
ordinal: 222000
card: "potiondrink"
owns:
  - "internal/combat/fighter.go"
  - "internal/resource/potion.go"
  - "internal/combat/potioneffect.go"
  - "internal/combat/potioneffect_test.go"
  - "internal/env/tuning_combat.go"
  - "docs/potions.md"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "bot tracks the working healing potion (tier, end, rate) and does not re-drink same or lower tier above combat.potionEmergencyPct (30); higher tier or emergency still drinks; tests on Sorceress and Paladin; verified offline only, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fid-potions.md: a second healing potion adds duration, not rate (V read, D arithmetic); the bot re-drinks every PotionGapMs below 55 percent. Done when: the pending heal is counted.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 pending heal estimate from tier, class and drink time
- [x] #2 no second healing potion while the pending heal covers the gap to the drink line
- [x] #3 unit test with two builds
<!-- AC:END -->

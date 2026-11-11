---
id: TASK-328
title: 'potiondrink: bot tracks a running potion before it drinks again'
status: Proposed
assignee:
  - pro-runner
created_date: '2026-11-11T19:21:00Z'
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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fid-potions.md: a second healing potion adds duration, not rate (V read, D arithmetic); the bot re-drinks every PotionGapMs below 55 percent. Done when: the pending heal is counted.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pending heal estimate from tier, class and drink time
- [ ] #2 no second healing potion while the pending heal covers the gap to the drink line
- [ ] #3 unit test with two builds
<!-- AC:END -->

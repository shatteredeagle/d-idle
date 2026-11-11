---
id: TASK-305
title: 'aggression2: pack hand-over keeps fighting, fight end reasons, pursuit on waypoint walk'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T15:56:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 211000
card: "aggression2"
owns:
  - "internal/combat/target.go"
  - "internal/combat/rank_shaman_test.go"
  - "docs/aggression.md"
  - "internal/bot/yield.go"
  - "internal/bot/yield_test.go"
  - "internal/combat/castfirst.go"
  - "internal/combat/dodge.go"
  - "internal/combat/dodge_test.go"
  - "internal/combat/fight.go"
  - "internal/nav/waypoint.go"
  - "internal/nav/pursuit_test.go"
exclusive: []
contracts: []
result: "Root cause V: packYield cleared once with a 4 s timeout (48 of 108 big fights ended at sec=4). Now repeats while the pack lives. Waypoint walk gets pursuit; end/remaining/life/mana on fight lines. Not verified live after deploy."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Paladin kills 2 to 3 of a pack and walks on. Done when: the hand-over fights the pack to the end and every fight logs why it ended.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 PackYield repeats the clear while the pack lives; end reasons logged; tests pass
<!-- AC:END -->

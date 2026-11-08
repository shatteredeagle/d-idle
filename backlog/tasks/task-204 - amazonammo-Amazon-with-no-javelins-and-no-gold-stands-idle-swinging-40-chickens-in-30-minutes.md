---
id: TASK-204
title: 'amazonammo: Amazon with no javelins and no gold stands idle swinging, 40 chickens in 30 minutes'
status: Proposed
assignee:
  - pro-runner
created_date: '2026-11-08T17:23:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: high
ordinal: 156000
card: "amazonammo"
owns:
  - "internal/combat"
  - "internal/leveler"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by panelslive (docs/soaks/classpacing2.md): lockstep Amazon seed 13 sat at clvl 3 and 158 kills from virtual minute 35 to 131 with gold 4, log 'combat weapon problem, not attacking' why 'thrown weapon stack empty' (weapon jav, ammo 0), then 'no bound attack skill, swinging the basic attack' 1251 times, 'combat abort fight: flee' 148 and 39 Save and Exit (chicken). Seeds 7 and 11 reached clvl 5 at 90 minutes. Equip also logs 'skip Buckler reason=unidentified'. Done when: the no-ammo Amazon recovers (melee, gold farm or a bow) instead of idling.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 An Amazon whose javelin stack is empty and gold is under a bow's price still gets XP (melee fallback, or a gold step) and does not loop chicken and town
- [ ] #2 Test with two classes (Amazon plus one non-thrower) for the no-ammo path
<!-- AC:END -->

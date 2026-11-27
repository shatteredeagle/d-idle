---
id: TASK-1023
title: 'act2temple: Act 2 quest dungeon life floors and crowd-on-arrival rule'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T20:00:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 486000
card: "act2temple"
lane: next
owns:
  - "internal/plandata/admission.json"
  - "docs/plan/data/admission.json"
  - "internal/admit/"
  - "internal/combat/arrival.go"
  - "internal/combat/arrival_test.go"
  - "internal/combat/safety.go"
  - "internal/combat/fighter.go"
  - "internal/combat/crowd.go"
exclusive: []
contracts: []
result: "Act 2 quest dungeon road rows now carry life floors (Sewers, Halls of the Dead, Claw Viper Temple, Maggot Lair, Palace Cellar, Ancient Tunnels; SME ballparks, S and U), and an arrival crowd row (14 hostiles within 8 subtiles in the first 60 s in the Halls, Claw Viper Temple and Maggot Lair levels) makes the fighter open a portal at once with no retreat walk. Unit tested, gate green on the builder, not verified live. The drill run on the old code showed 19 to 21 hostiles 12 s into the walk, not a kill 1 s after arrival; the run with the new rule is not done (the worker host stopped answering labrun). Build-specific: none, rows are data keyed by archetype."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Claw Viper Temple stand-in dies 1 s after arriving among 22 to 39 hostiles (matrix note 80). Done when the arrival row sends the hero back at once and the drill has a named outcome.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Road rows of the Act 2 quest dungeons carry level and life floors
- [x] #2 Arrival crowd rule as data, unit tested
- [ ] #3 drill.act2.amulet repeat 3 on the worker host
<!-- AC:END -->

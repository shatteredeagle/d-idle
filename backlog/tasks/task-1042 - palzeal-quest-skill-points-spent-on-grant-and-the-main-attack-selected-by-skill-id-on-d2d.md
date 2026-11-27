---
id: TASK-1042
title: 'palzeal: quest skill points spent on grant and the main attack selected by skill id on d2d'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T11:10:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 504000
card: "palzeal"
lane: next
owns:
  - "internal/leveler"
  - "internal/plandata/build-paladin-zealot-lite.json"
  - "docs/plan/data/build-paladin-zealot-lite.json"
  - "internal/combat/palzeal_attack_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Quest-tagged table points are bought only with quest bonus points, so a missing Den of Evil never shifts later rows; a granted bonus point is spent at once
- [ ] #2 Zealot-lite rows: clvl 4 Might, clvl 16 Zeal plus Holy Fire (radament), Den point tagged on row 5
- [ ] #3 On the d2d swinger backend the main attack is chosen by skill id from the profile band (Attack before Zeal is learned, Zeal from then), tested on Paladin and Necromancer Teeth
- [ ] #4 Live: a fresh zealot-lite Paladin logs skill=Zeal at kills from clvl 12
<!-- AC:END -->

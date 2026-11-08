---
id: TASK-143
title: 'classunlocks: per-class unlock files for the other six classes'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:03:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies:
  - TASK-132
priority: medium
ordinal: 132000
card: "classunlocks"
owns:
  - "internal/plandata/unlocks-amazon.json"
  - "internal/plandata/unlocks-assassin.json"
  - "internal/plandata/unlocks-barbarian.json"
  - "internal/plandata/unlocks-druid.json"
  - "internal/plandata/unlocks-necromancer.json"
  - "internal/plandata/unlocks-paladin.json"
  - "internal/plandata/unlocks_class_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 Amendment 2026-11-07, unlock schedules are per-class data. The Sorceress file is in TASK-133 (skillunlocks); this card does not duplicate it. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 unlocks-<class>.json for Amazon, Assassin, Barbarian, Druid, Necromancer and Paladin from section 2 of docs/research/class-early-game.md, ReqLevels as verified by TASK-132, UNSURE entries tagged; must-stay-free list per class (nothing the bot needs to survive locks later than Bishibosh)
- [ ] #2 A test loads every file and checks each class keeps its main attack, one defensive buff and an immunity skill free; unknown class is an error naming it; two-class test
- [ ] #3 go vet and go test -p 1 ./internal/plandata; leakcheck 0
<!-- AC:END -->

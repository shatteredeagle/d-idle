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
  - "docs/plan/data/unlocks-amazon.json"
  - "docs/plan/data/unlocks-assassin.json"
  - "docs/plan/data/unlocks-barbarian.json"
  - "docs/plan/data/unlocks-druid.json"
  - "docs/plan/data/unlocks-necromancer.json"
  - "docs/plan/data/unlocks-paladin.json"
  - "docs/skill-unlocks.md"
exclusive: []
contracts: []
result: "Six class files + tests passing offline; skill names and classes validated against the real tables on the worker host (V, offline). Event placements are the veteran schedule (S), not measured; deviations listed in docs/skill-unlocks.md. Not verified live. owns widened: docs/plan/data copies (TestMatchesDocs) and docs/skill-unlocks.md. Branch is stacked on agent/skillunlocks (needs internal/leveler/unlocks.go)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 Amendment 2026-11-07, unlock schedules are per-class data. The Sorceress file is in TASK-133 (skillunlocks); this card does not duplicate it. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 unlocks-<class>.json for Amazon, Assassin, Barbarian, Druid, Necromancer and Paladin from section 2 of docs/research/class-early-game.md, ReqLevels as verified by TASK-132, UNSURE entries tagged; must-stay-free list per class (nothing the bot needs to survive locks later than Bishibosh)
- [x] #2 A test loads every file and checks each class keeps its main attack, one defensive buff and an immunity skill free; unknown class is an error naming it; two-class test
- [x] #3 go vet and go test -p 1 ./internal/plandata; leakcheck 0
<!-- AC:END -->

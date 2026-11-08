---
id: TASK-146
title: 'classchores: per-class chore defaults for the policy sheet'
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
  - TASK-128
priority: medium
ordinal: 135000
card: "classchores"
owns:
  - "internal/plandata/chores-defaults.json"
  - "internal/plandata/chores_defaults_test.go"
  - "docs/classchores.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 Amendment 2026-11-07, chore defaults are per-class data. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Data rows keyed by class (all seven): potion mix, repair trigger, ammo trigger (Amazon bow only), summon upkeep, aura and buff choice, town-trip triggers, from section 4 of docs/research/class-early-game.md (EXP, tagged)
- [ ] #2 The policy sheet reads its starting defaults from the rows; unknown class is an error naming it; test with two classes (melee mana 0 potion mix vs a mana class)
- [ ] #3 go vet and go test -p 1 of the touched packages; leakcheck 0
<!-- AC:END -->

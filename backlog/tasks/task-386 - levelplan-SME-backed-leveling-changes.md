---
id: TASK-386
title: 'levelplan: SME-backed leveling changes'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-12T23:32:00Z'
updated_date: '2026-11-13T00:35:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 245000
card: "levelplan"
owns:
  - "docs/levelplan.md"
  - "internal/plandata"
  - "docs/plan/data/progression-normal.json"
  - "internal/loot"
  - "internal/env/tuning_loot.go"
  - "internal/bot/plan.go"
  - "internal/bot/plan_test.go"
exclusive: []
contracts: []
result: "Burial Grounds melee_min_clvl 8; exits Blood Moor 4, Cold Plains 8, Cave 1 9; loot.potionBuffer 4 + potionBufferMana 2; shrines proposal only (no shrine logic exists). Tests pass on the worker host, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 melee_min_clvl 8 for Burial Grounds, tests
- [x] #2 potion buffer 4 healing / 2 mana, tests
- [x] #3 early area exits per SME, doc table
- [x] #4 shrine finding and proposal in docs/levelplan.md
<!-- AC:END -->

---
id: TASK-266
title: 'beltplantest: belt plan shopping tests fail with real tables (mana buy rule vs player plan)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T18:40:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 196000
result: "Cause: TASK-240 ManaBuy rule blocked player-chosen mana columns (Barbarian need 0 never buys; Paladin needs HPFirst healing, unreachable on a small belt). Fixed with ManaBuyRule.ForPlan. Verified offline with real tables (D2_MPQ_DIR), not live."
card: "beltplantest"
owns:
  - "internal/town/shop.go"
  - "internal/resource/poverty.go"
  - "internal/resource/poverty_budget_test.go"
  - "internal/town/sim/env.go"
  - "docs/data.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 TestBeltPlanShoppingList passes with real tables
- [x] #2 ForPlan unit test covers Barbarian, Paladin, Sorceress
<!-- AC:END -->

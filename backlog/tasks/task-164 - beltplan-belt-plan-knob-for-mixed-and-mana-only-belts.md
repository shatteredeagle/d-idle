---
id: TASK-164
title: 'beltplan: belt plan knob for mixed and mana-only belts'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:18:00Z'
labels:
  - bot
  - feature
  - G-C
milestone: m-4
dependencies:
  - TASK-128
priority: medium
ordinal: 131000
card: "beltplan"
owns:
  - "internal/resource"
  - "internal/town"
  - "internal/env"
  - "internal/decide"
  - "internal/bot"
  - "internal/loot"
  - "docs/belt-plan.md"
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
- [ ] #1 belt plan templates and per-column overrides as data (resource.beltPlan, resource.beltCol1..4)
- [ ] #2 restock, belt fill, loot pickup and kit gate follow the plan
- [ ] #3 policy sheet rows potion mix and belt fill apply through the knob
- [ ] #4 table tests for each template, two classes, default regression, unknown template error
- [ ] #5 live drill all mana (or skipped with reason)
<!-- AC:END -->

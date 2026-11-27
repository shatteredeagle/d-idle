---
id: TASK-1074
title: 'mercaura: class merc aura from classtraits mercAura (HirePolicy takes the class)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T22:38:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 516000
card: "mercaura"
lane: next
owns:
  - "internal/d2/classtraits.go"
  - "internal/d2/classtraits.json"
  - "internal/d2/classtraits_test.go"
  - "docs/design/merc-aura-by-archetype.md"
  - "internal/town/merc.go"
  - "internal/town/merc_test.go"
  - "internal/task/act2.go"
  - "internal/task/act2_test.go"
  - "internal/bot/task.go"
  - "cmd/d2bot/merc_adapter.go"
exclusive: []
result: "Data-only: classtraits.json mercAura (Necromancer = today defaults per SME 2026-11-27), HirePolicy takes the class, Act 2 hire step reads it via task.MercAuraChooser. Unit tests only, NOT verified live; Sorceress/Paladin identical."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 mercAura per difficulty in classtraits.json, read by HirePolicy and the Act 2 hire step
- [x] #2 unknown class errors naming it; Sorceress and Paladin unchanged
<!-- AC:END -->

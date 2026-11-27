---
id: TASK-1063
title: 'classkeys: main-attack keys for every class, profile skills by name, merc aura note'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T20:14:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 512000
card: "classkeys"
lane: next
owns:
  - "internal/combat/classdata"
  - "internal/combat/profileskills_test.go"
  - "internal/plandata"
  - "docs/plan/data"
  - "cmd/d2bot/necroplay_test.go"
  - "cmd/d2bot/hotkeydrift_test.go"
  - "data/capabilities/capabilities.json"
  - "tools/provingground/scenarios/necro-epic.json"
  - "docs/design/merc-aura-by-archetype.md"
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
- [ ] #1 Every class's plan tables pass TestMainAttackSkillsHaveAKey and the hotkey ratchet is zero for Amazon, Barbarian, Druid, hammerdin
- [ ] #2 Every skill a profile names exists in the skills table for its class (test)
- [ ] #3 Merc aura by archetype design note
<!-- AC:END -->

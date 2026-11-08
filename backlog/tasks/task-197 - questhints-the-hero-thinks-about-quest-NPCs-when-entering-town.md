---
id: TASK-197
title: 'questhints: the hero thinks about quest NPCs when entering town'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T17:06:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies: []
priority: high
ordinal: 158000
card: "questhints"
owns:
  - "data/quests/act1-hints.json"
  - "data/quests/quests.go"
  - "internal/watch/questhints.go"
  - "internal/watch/questhints_test.go"
  - "docs/quest-hints.md"
  - "internal/decide/policy_chores.json"
  - "internal/decide/catalogue.json"
  - "internal/watch/cardtext.go"
  - "internal/plandata/chores-defaults.json"
  - "docs/plan/data/chores-defaults.json"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the hero should show quest NPCs in town like a player would. Done when: tests pass and the live drill card exists.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Hint table data with verified tags, family per class, unknown class error
- [ ] #2 Town-entry trigger: go once, weak re-arm after +2 levels, restart-safe
- [ ] #3 Policy row questhints (Quiet/Hints/Hints and ask me), surfaces in feed, Chronicle, return screen
- [ ] #4 Tests for Paladin, Sorceress, summoner; docs/quest-hints.md
<!-- AC:END -->

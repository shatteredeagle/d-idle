---
id: TASK-197
title: 'questhints: the hero thinks about quest NPCs when entering town'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T17:06:00Z'
updated_date: '2026-11-08T17:30:00Z'
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
  - "internal/decide/default_rules.json"
  - "internal/decide/parity_test.go"
  - "internal/decide/build_test.go"
  - "internal/watch/humanize.go"
  - "internal/watch/funwire.go"
  - "internal/watch/unlockchronicle.go"
  - "internal/watch/unlockvoice.go"
  - "internal/watch/testdata/policysheet"
exclusive: []
contracts: []
result: "Hint table (data/quests/act1-hints.json, tags SRC/EXP/UNSURE), QuestHints desk on the town-entry event, policy row questhints (Hints default for class starts, Quiet for named runs), two question points t1.quest.imbue and t1.quest.cain, feed, Chronicle and return-screen surfaces; tests for paladin, sorceress, necromancer. Offline only, NOT verified live; drill TASK-198."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the hero should show quest NPCs in town like a player would. Done when: tests pass and the live drill card exists.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Hint table data with verified tags, family per class, unknown class error
- [x] #2 Town-entry trigger: go once, weak re-arm after +2 levels, restart-safe
- [x] #3 Policy row questhints (Quiet/Hints/Hints and ask me), surfaces in feed, Chronicle, return screen
- [x] #4 Tests for Paladin, Sorceress, summoner; docs/quest-hints.md
<!-- AC:END -->

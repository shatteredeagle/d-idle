---
id: TASK-243
title: 'questgivers: the hero visits quest givers in town when a quest is available'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T01:43:00Z'
labels:
  - bot
  - feature
  - T4
dependencies: []
priority: medium
ordinal: 181000
card: "questgivers"
owns:
  - "internal/task/givers.go"
  - "internal/task/givers_test.go"
  - "internal/bot/givers.go"
  - "data/quests/act1-givers.json"
  - "docs/quest-givers.md"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. Table + pure decision + Campaign opt-in + tests for Paladin and Sorceress; live checklist in docs/quest-givers.md; also touched outside owns, tiny: internal/task/campaign.go (3 fields, one early return), internal/talk/tables.go (one row), internal/bot/runner.go (one wrap), data/quests/quests.go (one embed)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Slice B of the friends beta (user choice 2026-11-09). Done when the decision logic, table and fixtures are merged and the live checklist exists.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Table-driven quest givers (data/quests/act1-givers.json joined to the hint table), class-generic, unknown class is an error naming it; tests for Paladin and Sorceress
- [x] #2 Campaign visits the giver in town when the quest is not given, available and the family level is reached; no repeat per session, Normal only, opt-in; live part marked U with a checklist in docs/quest-givers.md
<!-- AC:END -->

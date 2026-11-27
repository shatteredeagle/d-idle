---
id: TASK-1043
title: 'questpts: every build table tags its quest bonus points'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T11:54:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 505000
card: "questpts"
lane: next
owns:
  - "internal/plandata"
  - "internal/buildfile"
  - "internal/leveler"
  - "docs/plan/data"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

PALZEAL follow-up: the quest key on build-table skill entries (bought only with quest bonus points) is applied to every table, not only the Zealot-lite one. Done when all tables are tagged and the validation test passes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every build table that folds a Den of Evil, Radament or Izual skill point into a level row tags it with a quest key; untagged rows hold one level point per level
- [ ] #2 buildfile expansion writes quest keys for the points its quest schedule pays
- [ ] #3 A test fails when a table holds more untagged points than its levels grant (all tables, all classes)
<!-- AC:END -->

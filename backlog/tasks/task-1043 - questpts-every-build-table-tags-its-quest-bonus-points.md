---
id: TASK-1043
title: 'questpts: every build table tags its quest bonus points'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T11:54:00Z'
updated_date: '2026-11-27T12:17:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 505000
card: "questpts"
lane: next
owns:
  - "internal/plandata/"
  - "internal/buildfile/"
  - "internal/leveler/"
  - "docs/plan/data/"
  - "internal/scenario/hero/"
exclusive: []
contracts: []
result: "24 build tables tag their folded Den of Evil, Radament and Izual points with a quest key (all endgame90, normal30, guide and fallback tables; the amazon bow, javelin, paladin-normal30, primary and fallback v1 tables fold none); buildfile expansion emits the keys; TestBuildTablesTagQuestPoints guards every table; hero prep and test replays pass quest points to Plan. Offline only, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

PALZEAL follow-up: the quest key on build-table skill entries (bought only with quest bonus points) is applied to every table, not only the Zealot-lite one. Done when all tables are tagged and the validation test passes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every build table that folds a Den of Evil, Radament or Izual skill point into a level row tags it with a quest key; untagged rows hold one level point per level
- [x] #2 buildfile expansion writes quest keys for the points its quest schedule pays
- [x] #3 A test fails when a table holds more untagged points than its levels grant (all tables, all classes)
<!-- AC:END -->

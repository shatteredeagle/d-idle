---
id: TASK-524
title: >-
  lootwire-L3: adapter filling lootrules.Item and Ctx from ground, pack, stash,
  hero and merc
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T02:11:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-523
priority: medium
ordinal: 284000
card: "lootwireL3"
owns:
  - "internal/loot/rulesadapt.go"
  - "internal/loot/rulesadapt_test.go"
  - "internal/loot/rulesfix_test.go"
exclusive: []
contracts: []
result: "loot.RulesAdapter: Item, ActorsOf, Ctx fill lootrules inputs from ground (d2d label flags), pack and stash units, hero and merc, needs and owned counts; unreadable fields stay hidden. Tested on a synthetic item table (no game files), two classes, two hirelings. Not verified live; stat ids for mana, frw, ias, gf, ed and durability are U."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the engine needs real item facts and a context. Done when: the adapter fills them from the game structs, unreadable fields flagged. Consumer: lootwire-L4.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ground, pack and stash items, hero and merc actors, fullness and Needs filled; unreadable fields stay hidden
- [x] #2 tests on a synthetic item table for two classes and two hirelings
<!-- AC:END -->

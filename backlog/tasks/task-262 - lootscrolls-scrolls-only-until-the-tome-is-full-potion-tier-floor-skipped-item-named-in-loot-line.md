---
id: TASK-262
title: >-
  lootscrolls: scrolls only until the tome is full, potion tier floor, skipped
  item named in loot line
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T18:32:00Z'
updated_date: '2026-11-10T18:48:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 196000
card: "lootscrolls"
owns:
  - "internal/loot"
  - "docs/lootskip.md"
exclusive: []
contracts: []
result: "Offline only (tests on the worker host with /tmp/d2mpq symlinked to the user install), not verified live. Build-specific data: internal/loot/potionfloor.json (per difficulty, optional per class). TomeCap 20 constant."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Scroll pickup stops when the tome is full (Paladin and Sorceress tests)
- [x] #2 Potion tier floor is data (potionfloor.json), tested on two classes
- [x] #3 loot summary names skipped items
<!-- AC:END -->

---
id: TASK-399
title: 'sellskip: sell chore skipped, recovery in Blood Moor, revive gold spent first'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-14T10:16:00Z'
updated_date: '2026-11-14T10:41:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 248000
card: "sellskip"
owns:
  - "docs/sellskip.md"
  - "internal/town/sellskip_test.go"
  - "internal/bot/sellskip_test.go"
  - "internal/loot/sellskip_test.go"
  - "internal/loot/loot.go"
  - "internal/town/shop.go"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "internal/bot/resource.go"
  - "internal/bot/goal.go"
  - "internal/bot/contracts.go"
  - "cmd/d2bot/adapters.go"
  - "internal/env/tuning_town.go"
exclusive: []
contracts: []
result: "root cause V (log+code): loot kept plain armor for an empty slot below clvl 15 though the player decides gear, so nothing wore or sold it. Fixed; no-portal recovery picks the level band via waypoints; shop keeps the revive gold; town budget line. Offline only: regression tests pass with the game tables on the worker host; not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: hero14 analysis findings 4 and 5: the sell chore ran in 1 of 158 trips (unworn early-gear keeps), a no-portal clvl 14 hero recovered in Blood Moor, a tome and scrolls took the merc revive gold. Done when: fixes and tests below, docs/sellskip.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sell chore sells plain armor when the player decides gear (root cause fixed, regression tests for two classes)
- [x] #2 No-portal recovery farms the level band via waypoints, not Blood Moor at clvl 14
- [x] #3 Shop keeps the merc revive gold from optional buys; town budget log line per visit
- [x] #4 docs/sellskip.md with V/U evidence
<!-- AC:END -->

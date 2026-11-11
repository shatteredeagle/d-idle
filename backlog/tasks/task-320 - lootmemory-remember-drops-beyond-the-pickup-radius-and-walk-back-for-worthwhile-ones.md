---
id: TASK-320
title: 'lootmemory: remember drops beyond the pickup radius and walk back for worthwhile ones'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T18:55:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 218000
card: "lootmemory"
owns:
  - "internal/loot/memory.go"
  - "internal/loot/memory_test.go"
  - "internal/loot/export_test.go"
  - "internal/loot/loot.go"
  - "internal/loot/pickup.go"
  - "internal/env/tuning_loot.go"
  - "internal/bot/goal.go"
  - "internal/bot/observe.go"
  - "docs/lootmemory.md"
  - "docs/knobs.md"
exclusive: []
result: "Built and tested offline on Sorceress and Paladin (sim, game tables on the worker host); NOT verified live. Build-specific: nothing (class-generic, knobs loot.mem*)."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: drops beyond the 30-subtile pickup radius (pack chases, legs, flights) were forgotten. Done when: they are remembered per game and fetched at quiet moments (docs/lootmemory.md).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Per-game loot memory records wanted drops (gold/items the pickit takes), forgets picked/gone/trashed ones, caps and evicts, resets on a new game
- [x] #2 WalkBack fetches remembered drops beyond the pickup radius at quiet moments, nearest first, limited by memMinGold, memMaxDist, memMaxTrips; aborts on a hostile; offline tests on Sorceress and Paladin
- [x] #3 Knobs loot.memEnabled/memMinGold/memMaxDist/memMaxTrips/memCap in docs/knobs.md; docs/lootmemory.md written (V/S/U)
- [ ] #4 Live check: after a pack fight on d2d, the hero walks back for a pile dropped beyond 30 subtiles (loot_memory log line)
<!-- AC:END -->

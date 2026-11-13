---
id: TASK-390
title: 'palprofile2: Paladin Sacrifice band and Holy Fire crowd rule'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T23:48:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 245000
card: palprofile2
owns:
  - "internal/combat/classdata"
  - "internal/combat/lifeprice.go"
  - "docs/palprofile.md"
exclusive: []
contracts: []
result: Sacrifice in the Paladin bands (default on, gated by life floor 70 with hysteresis, potions, merc) and Holy Fire only with 3+ hostiles close; unit-tested offline, not verified live (checklist in docs/palprofile.md).
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: decomp13 PALSKILL findings B1 (Sacrifice never cast) and B2 (Holy Fire before Might); data rows plus a life/potion/merc gate; see docs/palprofile.md Round 2. result: Sacrifice band and Holy Fire crowd rule built and unit-tested, default ON, NOT verified live (checklist in docs/palprofile.md).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sacrifice in the Paladin attack bands behind combat.sacrifice with life floor, potion and merc gates
- [x] #2 Holy Fire only with combat.holyFireMinMonsters hostiles close, Might otherwise
- [x] #3 tests: bands per clvl (Paladin, Amazon), never under the floor, aura by monster count
- [x] #4 docs/palprofile.md with V/U and the sandbox live-check list
<!-- AC:END -->

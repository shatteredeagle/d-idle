---
id: TASK-68
title: 'g2gfixscroll: no town portal scroll with funded gold leaves the hero stuck outside'
status: Done
assignee: []
created_date: '2026-11-07T07:05:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - bot-defect
dependencies: []
priority: medium
ordinal: 66000
card: "g2gfixscroll"
owns:
  - "internal/town/chores.go"
  - "internal/bot/leveling.go"
  - "internal/leveler/equip.go"
  - "internal/leveler/leveler.go"
  - "internal/leveler/shop.go"
  - "internal/env/tuning_town.go"
  - "cmd/d2drill/e1_tp_first.go"
  - "cmd/d2drill/e1_equip.go"
  - "docs/probes/shortdrills.md"
exclusive: []
contracts: []
result: "bot fix: gearStep restocks the portal scroll (Townsman Chores) before the gear shop when TPCount < town.tpReserve and gold covers one; a failed equip is remembered by item kind (town.equipKindFails). Live drill E1.tp-first-a/b PASS on slot 4, writes after start 0; E1.equip-fail INCONCLUSIVE (the d2d accepted every swap). Equip to leave-game causal link not found in code (save point and fresh game are separate paths). Offline tests: bot TestScrollIsBoughtBeforeGear, town TestRestockBeforeGearBuysTheMissingScroll, leveler TestEquipFailureMemoryByKindAcrossGames."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/research/g2g-diagnosis.md finding S1: seed 17 minute 1035 to 1050, no town portal scroll at gold 304 to 460, pocket stall in Black Marsh, 15 virtual minutes without a kill. U on why the scroll was not bought.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 town code buys a scroll before gear when the scroll count is 0 and gold covers it
- [x] #2 offline test with a fake shop
<!-- AC:END -->

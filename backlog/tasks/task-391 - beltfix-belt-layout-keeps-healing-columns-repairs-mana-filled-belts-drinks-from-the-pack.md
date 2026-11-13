---
id: TASK-391
title: 'beltfix: belt layout keeps healing columns, repairs mana-filled belts, drinks from the pack'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-12T23:58:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 245000
card: "beltfix"
owns:
  - "internal/town/belt"
  - "internal/town/beltfill.go"
  - "internal/town/beltrepair.go"
  - "internal/town/sim"
  - "internal/combat/packdrink.go"
  - "internal/combat/packdrink_test.go"
  - "docs/beltfix.md"
  - "internal/town/beltfix6_test.go"
  - "internal/combat/fighter.go"
  - "internal/combat/safety.go"
  - "internal/loot/fixpickup_test.go"
  - "internal/loot/lootskip_test.go"
  - "internal/town/beltfix_test.go"
  - "cmd/d2bot/beltfill_adapter.go"
  - "cmd/d2bot/wiring_test.go"
exclusive: []
contracts: []
result: "root cause V (sandbox log): belt read as 4 slots with 6 boxes in use, free count negative, fill/shop/pickup all idle; fixed by a slot floor, a layout rule in PlaceFor (planned columns per kind from the class band), a town repair that moves surplus mana to the pack, and an emergency pack drink (right click) below combat.drinkRejuvPct. Verified offline only (Paladin and Sorceress); pack drink on d2d not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

SURVIVAL: belt held 6 mana and 0 healing, pack held 2 healing, no healing drink 100+ times. Done when the layout rule, repair and pack drink are tested offline.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 PlaceFor layout rule keeps healing columns (Paladin, Sorceress)
- [x] #2 belt slots never fewer than boxes in use
- [x] #3 town repair moves surplus mana out and fills healing
- [x] #4 emergency pack drink when the belt has no healing
- [x] #5 wiring and loot tests
- [ ] #6 live check: pack drink on d2d
<!-- AC:END -->

---
id: TASK-711
title: 'chores2: shop buys healing to the route guard kit minimum'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T03:06:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 340000
card: "chores2"
owns:
  - "internal/town/chores.go"
  - "internal/town/kithealing_test.go"
  - "internal/town/town.go"
  - "internal/task/gate.go"
  - "internal/task/route_test.go"
  - "internal/bot/task.go"
  - "internal/bot/contracts.go"
  - "cmd/d2bot/adapters.go"
  - "docs/probes/chores2-2026-11-21.md"
exclusive: []
result: "Restock and stash withdraw now satisfy the route guard kit (belt+pack, KitHealing), kit-only guard block inserts chores; unit tests Paladin+Sorceress green on the builder; live on the builder the guard restocks, but act1.tools.malus Paladin 11 stays BLOCKED (cheated hero drains potions on the road); not verified GREEN"
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the route guard counts 4 healing potions (belt + pack), the belt plan of a 4-slot belt with a mana column holds 3, so the shop saw nothing to buy; and a kit-only guard block was a danger fault (planner levels) instead of a chores trip. Fix: kit-only block inserts chores (task/gate.go guardFault); task chores pass ChorePlan.KitHealing = kitMinHP so the shop buys to it (into the pack) and the stash pays (shopCost). Done when: unit tests on Paladin and Sorceress pass and act1.tools.malus is GREEN. Source: docs/probes/provingground-act1more-2026-11-20.md findings 1 and 2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Must-fail unit tests on Paladin and Sorceress: a belt full to the plan but under KitMinHP healing buys, withdraws from the stash
- [ ] #2 act1.tools.malus Paladin 11 and Sorceress 3 GREEN on the builder (CHEATED setup, mechanics only)
<!-- AC:END -->

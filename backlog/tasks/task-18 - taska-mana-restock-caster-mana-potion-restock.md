---
id: TASK-18
title: "TASK-A mana-restock: caster mana potion restock"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-B"
dependencies: []
priority: high
ordinal: 18000
card: "taska"
owns:
  - "internal/town/shop.go"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "internal/town/manarestock_test.go"
  - "internal/town/earlygear_test.go"
  - "internal/town/town_test.go"
  - "internal/env/tuning_route.go"
  - "internal/task/route.go"
  - "internal/task/world.go"
  - "internal/task/towerlife_test.go"
  - "internal/task/floors_test.go"
  - "internal/bot/routeguard.go"
  - "internal/bot/task.go"
  - "internal/bot/towerguard_test.go"
  - "internal/bot/routeguard_test.go"
  - "docs/knobs.md"
  - "backlog/tasks/task-18 - taska-mana-restock-caster-mana-potion-restock.md"
exclusive: []
contracts:
  - "early-game-policy"
branch: "agent/taska-manarestock"
agent: "taska"
result: "Offline tested (go test -p 1 ./internal/town ./internal/resource ./internal/bot on the worker host); NOT run live."
live: "not run; proof line in Implementation Notes"
merge_hash: ""
overlap: "docs/research/lvl14plan.md TASK-A named internal/resource/beltplan.go; the cause turned out to be the town shop (the resource planner already buys MPFirst), so the change is in internal/town."
source: "docs/research/lvl14plan.md TASK-A"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Gate 2 rerun seed 11 (clvl 4 to 16) fought with belt mp = 0 for hours (`combat out of mana: backing off` 2,400+ times) and Tamoe Highland was route-guarded for `1 mana potions < 2 (caster)`.

Root cause (V, code read): `Town.buyPotions` (internal/town/shop.go) filled the kinds in the order HP, MP, RV with the full planned count of healing potions first, so a purse of a few hundred gold went into healing and `manaReserve` then refused the mana. `resource.PlanBuys` already has the HPFirst/MPFirst order but only the simulator calls it. Second gap: `tripReasons` raised a mana trip only at fewer than `town.tripMinMPPotions` (1) potions, below the route kit's 2.

Change: for a caster (`resource.ManaNeed` >= 2: Sorceress, Necromancer, Druid) the shop buys HPFirst healing, then MPFirst mana potions, then the normal fill; a caster at `resource.kitGateLevel` or above with gold for one potion also trips at fewer than MPFirst mana potions. Build-specific data: belt columns come from the class plan (`resource.PlanFor`), HPFirst/MPFirst/KitGateLevel are Resource knobs; a class without a mana column has want[MP] == 0 and shops as before. No number in a contract changes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 caster with gold ends a visit with the first mana potions and the healing plan filled (TestCasterBuysFirstManaBeforeHealingFill)
- [x] #2 caster with 20 gold buys nothing unaffordable and logs `caster: no mana, gold N` once per gold value (TestCasterPoorBuysNoManaAndLogs)
- [x] #3 Barbarian (no mana column) unchanged (TestMeleeBuildUnchangedByManaRestock, TestCasterTripAtFewerThanFirstMana)
- [x] #4 trip gating: gold for a potion required, class from ManaNeed
- [x] #5 go test -p 1 ./internal/town ./internal/resource ./internal/bot pass
- [ ] #6 live proof (Gate 2/3 run, below)
- [x] #7 TASK-D folded in (SME, user-approved 2026-11-04, numbers U): Forgotten Tower without a living merc needs clvl 15, max life 150 and the kit (4 healing, a portal with charges, 3 mana potions for a class with ManaNeed 2); with a living merc clvl 14; max life unreadable falls back to clvl 16 (TestTowerFloorLifeAndKit, TestTowerLifeLogLine, TestTowerLifeRuleThroughTheBot, TestFillTowerManaFollowsTheManaColumn)
- [ ] #8 live proof of the tower guard (Gate 2/3 run, below)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
result: offline tested, not run live.

Live proof line (Gate 2/3 log): `town: restocked mana potions bought=N mpBefore=0 mpAfter=N gold=G` after a town visit, and `shop: visit ... beltMP=N`; with no gold, `caster: no mana, gold N`. Expected: no run of `combat out of mana: backing off` with belt mp = 0 after a visit that had gold, and no `route guard (1 mana potions < 2 (caster))` once gold allows.

Two existing fixtures (TestNeedsTrip, TestFoundGearTripOnlyForNeededWeapon) held one mana potion as "stocked"; they now hold two, since a caster below MPFirst has its own trip reason.

Not verified (U): rig vendor prices are higher than the game's; the number of trips per hour with the new threshold (the existing trip budget in resource.TripBudgetSpent still applies upstream) needs a live run.

TASK-D (Forgotten Tower floor), folded into this task because the card cap (ADR 0015) allows no new open task. Old rule: `DefaultRouteTuning` TowerMinLevel 16 (internal/env/tuning_route.go), applied by `towerFloor` (internal/task/route.go). New knobs (defaults): `route.towerMinLevel` 15, `route.towerMinLife` 150 (0 = life rule off), `route.towerMinLevelNoLife` 16, `route.towerKitMana` 3, `route.towerMinLevelMerc` 14 unchanged. The merc is no longer required. Build-specific: all numbers are Route knobs; the mana part follows `resource.ManaNeed` >= 2 (Barbarian unchanged). Contract prose in merc-policy (lines about the Forgotten Tower floor 16) is owned by the m3-merc tasks and not edited here; proposed text: "Forgotten Tower: no merc clvl 15 and max life 150 and the kit, merc clvl 14, unreadable life clvl 16 (SME 2026-11-04, U)".

Live proof line (Gate 2/3 log): `route guard: forgotten tower floor=15 (merc=none)`, a closed guard logs `route guard: blocked tower (no merc, life 149 < 150)`; an unreadable life logs `route guard: forgotten tower floor=16 (merc=none) (max life unreadable, fallback floor)`. Not run live.
<!-- SECTION:NOTES:END -->

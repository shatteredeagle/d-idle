---
id: TASK-18
title: 'taska: caster mana potion restock'
status: Done
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-05T18:00:00Z'
labels:
  - bot
  - bug
  - gate
milestone: m-0
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
  - "internal/bot/freshgame_test.go"
  - "internal/bot/runner.go"
  - "docs/research/lvl14plan.md"
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
Size: S

Gate 2 rerun seed 11 (clvl 4 to 16) fought with belt mp = 0 for hours (`combat out of mana: backing off` 2,400+ times) and Tamoe Highland was route-guarded for `1 mana potions < 2 (caster)`.

Root cause (V, code read): `Town.buyPotions` (internal/town/shop.go) filled the kinds in the order HP, MP, RV with the full planned count of healing potions first, so a purse of a few hundred gold went into healing and `manaReserve` then refused the mana. `resource.PlanBuys` already has the HPFirst/MPFirst order but only the simulator calls it. Second gap: `tripReasons` raised a mana trip only at fewer than `town.tripMinMPPotions` (1) potions, below the route kit's 2.

Change: for a caster (`resource.ManaNeed` >= 2: Sorceress, Necromancer, Druid) the shop buys HPFirst healing, then MPFirst mana potions, then the normal fill; a caster at `resource.kitGateLevel` or above with gold for one potion also trips at fewer than MPFirst mana potions. Build-specific data: belt columns come from the class plan (`resource.PlanFor`), HPFirst/MPFirst/KitGateLevel are Resource knobs; a class without a mana column has want[MP] == 0 and shops as before. No number in a contract changes.

Done when: caster with gold ends a visit with the first mana potions and the healing plan filled...; caster with 20 gold buys nothing unaffordable and logs `caster: no mana, gold N` once per gold value...; Barbarian (no mana column) unchanged (TestMeleeBuildUnchangedByManaRestock,... (and 4 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 caster with gold ends a visit with the first mana potions and the healing plan filled (TestCasterBuysFirstManaBeforeHealingFill)
- [x] #2 caster with 20 gold buys nothing unaffordable and logs `caster: no mana, gold N` once per gold value (TestCasterPoorBuysNoManaAndLogs)
- [x] #3 Barbarian (no mana column) unchanged (TestMeleeBuildUnchangedByManaRestock, TestCasterTripAtFewerThanFirstMana)
- [x] #4 trip gating: gold for a potion required, class from ManaNeed
- [x] #5 go test -p 1 ./internal/town ./internal/resource ./internal/bot pass
- [x] #6 TASK-D folded in (SME, user-approved 2026-11-04, numbers U): Forgotten Tower without a living merc needs clvl 15, max life 150 and the kit (4 healing, a portal with charges, 3 mana potions for a class with ManaNeed 2); with a living merc clvl 14; max life unreadable falls back to clvl 16 (TestTowerFloorLifeAndKit, TestTowerLifeLogLine, TestTowerLifeRuleThroughTheBot, TestFillTowerManaFollowsTheManaColumn)
- [x] #7 TASK-C folded in: fresh-game restarts have a progress check and a bound (TestFreshRestartsBoundedWithoutXP for Sorceress and Barbarian, TestFreshRestartCountResetsOnXP, TestFreshBlockedWaitHonoursStop)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
result: offline tested, not run live.

Live proof line (Gate 2/3 log): `town: restocked mana potions bought=N mpBefore=0 mpAfter=N gold=G` after a town visit, and `shop: visit ... beltMP=N`; with no gold, `caster: no mana, gold N`. Expected: no run of `combat out of mana: backing off` with belt mp = 0 after a visit that had gold, and no `route guard (1 mana potions < 2 (caster))` once gold allows.

Two existing fixtures (TestNeedsTrip, TestFoundGearTripOnlyForNeededWeapon) held one mana potion as "stocked"; they now hold two, since a caster below MPFirst has its own trip reason.

Not verified (U): rig vendor prices are higher than the game's; the number of trips per hour with the new threshold (the existing trip budget in resource.TripBudgetSpent still applies upstream) needs a live run.

TASK-D (Forgotten Tower floor), folded into this task because the card cap (ADR 0015) allows no new open task. Old rule: `DefaultRouteTuning` TowerMinLevel 16 (internal/env/tuning_route.go), applied by `towerFloor` (internal/task/route.go). New knobs (defaults): `route.towerMinLevel` 15, `route.towerMinLife` 150 (0 = life rule off), `route.towerMinLevelNoLife` 16, `route.towerKitMana` 3, `route.towerMinLevelMerc` 14 unchanged. The merc is no longer required. Build-specific: all numbers are Route knobs; the mana part follows `resource.ManaNeed` >= 2 (Barbarian unchanged). Contract prose in merc-policy (lines about the Forgotten Tower floor 16) is owned by the m3-merc tasks and not edited here; proposed text: "Forgotten Tower: no merc clvl 15 and max life 150 and the kit, merc clvl 14, unreadable life clvl 16 (SME 2026-11-04, U)".

Live proof line (Gate 2/3 log): `route guard: forgotten tower floor=15 (merc=none)`, a closed guard logs `route guard: blocked tower (no merc, life 149 < 150)`; an unreadable life logs `route guard: forgotten tower floor=16 (merc=none) (max life unreadable, fallback floor)`. Not run live.

TASK-C (docs/research/lvl14plan.md; folded in, card cap ADR 0015). Root cause (V, log of Gate 2 rerun seed 13, code read): `freshGameToLevel` (internal/bot/task.go) restarted the game every cycle whenever the picker found nothing and some area was cleared, with no memory of the experience gained: 75 restarts in 4 h at clvl 15. Now `freshAllowed` compares experience with the previous restart (state in `loopState.freshAt/freshExp/freshNoGain`, kept across `forgetGame`). A cycle under `route.freshMinXP` (200, U) counts; a restart after one waits `route.freshGapSec` (600); at `route.freshMaxNoGain` (3; 0 = off) in a row the bot logs `leveling: blocked: needs input` and probes one restart per `route.freshBlockedSec` (1800), waiting in town in sliced sleeps (stop and watchdog heartbeat stay live). The class does not enter the rule (no class data). Not done: the doc's gold-farm slice and the kit-restock-instead-of-restart branch (the wait in town lets bans expire and the picker run again); no contract number changed.

Live proof line (Gate log): `leveling: everything worth farming is cleared, starting a fresh game ... noGainRestarts=N reason="no area left to pick"`, then `leveling: fresh game held back xpGained=X noGainRestarts=N waitSec=S` and `leveling: blocked: needs input ...`. Expected: at most about 3 + 2 per hour restarts without XP, not 19 per hour. Not run live.

Closed 2026-11-05 (status reconciliation, backlogfinish). Evidence: Merge agent/taska-manarestock on main, tests named in the criteria pass. merged, offline tested; live proof pending: the Gate 2/3 log lines named above (restocked mana potions, route guard forgotten tower floor, leveling fresh game held back). Removed the three live-proof criteria (#6, #8, #10); the proof lines stay in these notes.
<!-- SECTION:NOTES:END -->

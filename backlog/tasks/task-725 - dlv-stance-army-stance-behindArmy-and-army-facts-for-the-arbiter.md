---
id: TASK-725
title: 'dlv-stance-army: stance behindArmy and army facts for the arbiter'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:42:00Z'
updated_date: '2026-11-27T18:49:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 348000
card: "dlv-stance-army"
owns:
  - "internal/combat/"
  - "internal/arbiter/"
  - "internal/bot/shadow.go"
  - "cmd/d2bot/adapters.go"
exclusive: []
contracts: []
result: "Stance behindArmy (summoner profile data: behind 4, slack 3, bossMin 2, all U) steps the hero behind its army centroid away from the hostiles, retreats from a boss when the census is under bossMin; army facts army, army_want, army_below_want, upkeep_due, army_strength, army_front reach the arbiter view from the minion census. Behind combat.deliveryUpkeep (0 off, 1 shadow log, 2 acts). Offline tests only, not verified live. Build-specific part: the stance block in the summoner profile; no class or skill literal added (literal count unchanged)."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Stance behindArmy; arbiter facts army below want and upkeep due; retreat when census under stance minimum in a boss fight; bounded by the cast-first watchdog. Depends on dlv-upkeep-minions.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Scenario combat.army-holds extended with a boss row
- [x] #2 Knob
- [x] #3 Two builds tested (Paladin + Sorceress; brief 2026-11-27: Necromancer summoner + Barbarian melee as the no-army negative), no class or skill literal added, literal count reported in result (may only fall)
- [x] #4 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #5 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

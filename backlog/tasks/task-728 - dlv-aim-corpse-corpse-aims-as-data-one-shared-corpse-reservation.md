---
id: TASK-728
title: 'dlv-aim-corpse: corpse aims as data, one shared corpse reservation'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:43:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 351000
card: "dlv-aim-corpse"
owns:
  - "internal/combat/"
exclusive: []
contracts: []
lane: next
result: "Corpse aims as profile rows (Profile.CorpseAims: corpseNearest, corpseBest, corpseNearPack with radius and minHit, tagged U) in internal/combat/aimcorpse.go, which replaces necro_corpse.go; rows shipped for Corpse Explosion and Revive on the three Necromancer profiles. One corpse reservation (corpseRes, takeCorpse) now shared by raise, revive, explode and Death Sentry. Knob combat.deliveryUpkeep 0/1/2 (0 and 1 pick as the old callers did, 1 counts rows that would differ, 2 rows pick and may refuse); counters via Fighter.CorpseCounts. Offline tests: Necromancer rows and modes, Sorceress with no rows keeps the default, validation. Find Potion and Find Item utility rows not done. No new Go skill or class literal. Scenario combat.corpse-explodes not written (live); not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). corpseBest, corpseNearPack as data; one corpse reservation shared by raise, revive, explode, Death Sentry; Find Potion and Find Item as utility rows; removes necro_corpse.go. Depends on dlv-upkeep-minions, dlv-cast-verbs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenario combat.corpse-explodes
- [x] #2 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #3 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [ ] #4 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

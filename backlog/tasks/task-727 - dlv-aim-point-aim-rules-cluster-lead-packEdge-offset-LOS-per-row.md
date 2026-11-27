---
id: TASK-727
title: 'dlv-aim-point: aim rules cluster, lead, packEdge, offset; LOS per row'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:43:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 350000
card: "dlv-aim-point"
owns:
  - "internal/combat/"
exclusive: []
contracts: []
lane: next
result: "Point aim rows as profile data (cluster, lead, packEdge, offset, crowd gate, line of sight missile, walk or none) in internal/combat/aim.go; rows shipped for the Sorceress Blizzard and Druid Eruption (Fissure) profiles, all radii tagged U; wired into the main cast, the watchdog (never crowd-gated) and the Assassin trap lay behind the existing knob combat.deliveryUpkeep 0/1/2 (0 unchanged, 1 shadow counts, 2 aims); counters via Fighter.AimCounts. Offline tests on Sorceress, Druid and Paladin (melee, never re-aimed; aim rows on melee skills fail validation). Build-specific data lives only in profiles.json; Go skill and class literal count in internal/combat unchanged at 108. No trap rows yet, so combat.traps-stand is open. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Aim rules from table data; point scoring as a gate (crowdAtLeast); line of sight per row (missile, walk, none) also on the watchdog path; traps on the same path. Depends on dlv-cast-verbs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenarios combat.point-spell-hits (Druid Fissure, Sorceress Blizzard) and combat.traps-stand
- [ ] #2 refused_at_most assertion so a fixed fork does not break it
- [x] #3 No tier rise before fidelity A5 and A8 verdicts
- [x] #4 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #5 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #6 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

---
id: TASK-723
title: 'dlv-upkeep-minions: minion-observed upkeep: summons, golems, pets'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-21T04:42:00Z'
updated_date: '2026-11-27T18:49:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 346000
card: "dlv-upkeep-minions"
owns:
  - "internal/combat/"
exclusive: []
contracts: []
result: "Offline only, not verified live. A census-driven minion driver (minions.go) keeps summon groups from profile data (necro block derived, druid and assassin minions blocks), behind combat.deliveryUpkeep (1 shadow, 2 acts). Offline scenarios army-holds and pets-hold pass; the shadow plays the old Necromancer loop with zero disagreements. Literal count in the summon files 35 to 33. The old class loops stay until knob 2 is proven; differential A11 and the live shadow run are open. Remaining (live): shadow run versus the old loop and differential A11 (ACs 2 and 3). Remaining (live): AC 2 shadow run vs the old loop and AC 3 differential A11 (combat.army-holds, combat.pets-hold on the fork build)."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Groups from the summon column, want from pet cap, corpse-target summons via corpse aim rules; deletes necro.go summon planning, druid.go plans, assassin.go shadow code, class switch. Must first reproduce the Necromancer loop in shadow (knob 1). Experimental skills enter the rotation behind the expectation breaker, unsupported never (Q6). Depends on dlv-upkeep-state, dlv-reads.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Scenarios combat.army-holds (Necromancer) and combat.pets-hold (Druid) first
- [ ] #2 Shadow run matches the old loop
- [ ] #3 Differential A11 for specials
- [x] #4 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #5 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #6 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

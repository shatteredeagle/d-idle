---
id: TASK-732
title: 'dlv-move-melee: move driver part 2: Charge and Leap Attack'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-21T04:44:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 355000
card: "dlv-move-melee"
owns:
  - "internal/combat/"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. Gap rows in the move driver (move.go gapClose): Leap Attack (Barbarian default and frenzy; gap through the target by 1 subtile, plus escape out of a surround) and Charge (Paladin default and hammerdin; gap at the target), numbers are planning values (source U). A left-hand skill is refused unless the backend casts by id (hotkeys select only the right hand), so Charge needs d2d. Knob combat.deliveryUpkeep 0/1/2. Tests Barbarian + Paladin (gap) and Sorceress (hop). No layout keys added. Live: the combat.teleport-reposition rows and the landing rules of Leap Attack and Charge. Remaining (live): AC 4 race gate x20 on the worker host; Leap Attack and Charge rows live in combat.teleport-reposition."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Charge and Leap Attack as gap closers (aim target and through), Leap as an escape. Depends on dlv-move-tele, dlv-press.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Row added to combat.teleport-reposition for a Barbarian and a Paladin
- [x] #2 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #3 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [ ] #4 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

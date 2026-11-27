---
id: TASK-734
title: 'dlv-whirl: Whirlwind: channel + through + commit (PARKED)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:45:00Z'
labels:
  - bot
  - feature
  - parked
dependencies: []
priority: medium
ordinal: 356000
card: "dlv-whirl"
owns:
  - "internal/combat/"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. Whirlwind driver: move use whirl (move.go rows, whirl.go leg planner and whirlTick after moveTick): one leg per fight tick aimed through the densest pack within range, prefers another pack than the last leg, no leg under whirlAbortLifePct (life floor), the spin is one press slice (press row, Barbarian default) so potions run between legs. Barbarian default profile carries the Whirlwind move and press rows (U planning values, spin radius waits for fidelity A6); acts only once Whirlwind is learned. Started before the A6 verdict on the coordinator brief (AC2 waived by the brief, not met). Knob combat.deliveryUpkeep 0/1/2, no new knob. No skill or class literal added in Go. Tests Barbarian plus a Paladin test row (Zeal). Live: scenario combat.whirlwind-clears and the drill commands in the task report."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). PARKED (Q9): Whirlwind and shapeshift forms wait until fidelity A6-A8 have verdicts. Channel + through + commit, path across the pack, no drink during the spin. Depends on dlv-move-melee.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenario combat.whirlwind-clears
- [ ] #2 Starts only after fidelity A6 verdict and dlv-move-melee
- [x] #3 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #4 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [ ] #5 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

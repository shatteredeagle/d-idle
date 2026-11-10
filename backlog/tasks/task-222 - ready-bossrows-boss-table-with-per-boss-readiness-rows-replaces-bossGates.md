---
id: TASK-222
title: 'ready-bossrows: boss table with per-boss readiness rows replaces bossGates'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - feature
  - fidelity
  - parked
dependencies:
  - TASK-214
priority: medium
ordinal: 171000
card: "ready-bossrows"
owns:
  - "internal/readiness/bossrows.go"
  - "internal/readiness/bossrows_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Per-boss rows for all act bosses from install data. Done when the bossGates map is retired.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Boss table for all act bosses with per-boss rows (resists, immunities, kit); Normal/NM/Hell values from install data; bossGates map retired
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
- [ ] #3 Open question settled first: whether Player.Resist already includes the difficulty penalty (TODO in internal/combat/hard.go) is taken from the ready-audit result and the resist logic here uses that answer; a test covers both possible cases of the ambiguity until it is settled
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: Normal Act 1 does not need the readiness rows (beta review section 8); revisit when a Nightmare character exists.
<!-- SECTION:NOTES:END -->

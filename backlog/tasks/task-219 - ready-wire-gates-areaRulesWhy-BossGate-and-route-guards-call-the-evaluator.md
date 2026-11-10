---
id: TASK-219
title: 'ready-wire-gates: areaRulesWhy, BossGate and route guards call the evaluator'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - bot-fix
  - fidelity
  - parked
dependencies:
  - TASK-214
priority: medium
ordinal: 168000
card: "ready-wire-gates"
owns:
  - "internal/bot/riskfacts.go"
  - "internal/bot/geargate.go"
  - "internal/task/gate.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). The bot's gates consume the same checks as the meter so they cannot disagree. Done when the gates call the evaluator and no verdict is looser than today.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 areaRulesWhy, BossGate, route guards call the evaluator; existing gate tests unchanged; new test: no verdict looser than today for the same inputs
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
- [ ] #3 Open question settled first: whether Player.Resist already includes the difficulty penalty (TODO in internal/combat/hard.go) is taken from the ready-audit result and the resist logic here uses that answer; a test covers both possible cases of the ambiguity until it is settled
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: Normal Act 1 does not need the readiness rows (beta review section 8); revisit when a Nightmare character exists.
<!-- SECTION:NOTES:END -->

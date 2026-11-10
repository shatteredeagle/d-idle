---
id: TASK-224
title: 'ready-calibrate: replay recorded deaths through the readiness evaluator'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-09T14:28:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - drill
  - fidelity
  - parked
dependencies:
  - TASK-214
priority: medium
ordinal: 173000
card: "ready-calibrate"
owns:
  - "docs/probes/ready-calibrate.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Thresholds start as U numbers. Done when the replay reports hits and false-reds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Replay deaths from docs/deathautopsy.md / death logs: the evaluator would have flagged red/amber before at least N of M recorded deaths (N, M set when the card is groomed); false-red count on survived sessions reported
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
- [ ] #3 Open question settled first: whether Player.Resist already includes the difficulty penalty (TODO in internal/combat/hard.go) is taken from the ready-audit result and the resist logic here uses that answer; a test covers both possible cases of the ambiguity until it is settled
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: Normal Act 1 does not need the readiness rows (beta review section 8); revisit when a Nightmare character exists.
<!-- SECTION:NOTES:END -->

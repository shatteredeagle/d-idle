---
id: TASK-220
title: 'ready-meter: risk meter rows are the readiness check set'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - feature
  - fidelity
  - parked
dependencies:
  - TASK-214
priority: medium
ordinal: 169000
card: "ready-meter"
owns:
  - "internal/watch/unlockrisk.go"
  - "internal/watch/unlockrisk_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). The watch risk meter shows each check as a row. Done when ResistGap is live and golden JSON exists for two classes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Risk meter rows are the check set; ResistGap live; golden JSON for two classes
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
- [ ] #3 Open question settled first: whether Player.Resist already includes the difficulty penalty (TODO in internal/combat/hard.go) is taken from the ready-audit result and the resist logic here uses that answer; a test covers both possible cases of the ambiguity until it is settled
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: same meter as TASK-159 (merged); revisit with TASK-219 when a Nightmare character exists.
<!-- SECTION:NOTES:END -->

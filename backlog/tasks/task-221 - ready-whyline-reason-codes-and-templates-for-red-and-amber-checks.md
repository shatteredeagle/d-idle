---
id: TASK-221
title: 'ready-whyline: reason codes and templates for red and amber checks'
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
ordinal: 170000
card: "ready-whyline"
owns:
  - "internal/bot/whyline.go"
  - "internal/bot/whyline_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Whyline explains why the planner holds back or retreats. Done when held-back steps show the reason.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Reason codes and templates (2 phrasings each) for red/amber checks; held-back planner steps show the reason
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: Normal Act 1 does not need the readiness rows (beta review section 8); revisit when a Nightmare character exists.
<!-- SECTION:NOTES:END -->

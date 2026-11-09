---
id: TASK-223
title: 'ready-policy: readiness strictness policy row (Strict, Balanced, Bold)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
labels:
  - watch
  - feature
  - fidelity
dependencies:
  - TASK-214
priority: medium
ordinal: 172000
card: "ready-policy"
owns:
  - "internal/watch/funwire.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Player-chosen margin shift like the stance row. Done when the row exists and the default changes nothing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 "Readiness strictness" policy row with per-class knob values; default changes nothing
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
<!-- AC:END -->

---
id: TASK-217
title: 'ready-area-table: generated per-area table of mlvl, immunities and elite affixes'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
labels:
  - bot
  - feature
  - fidelity
dependencies:
  - TASK-214
priority: medium
ordinal: 166000
card: "ready-area-table"
owns:
  - "internal/readiness/areatable.go"
  - "internal/readiness/areatable_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Per-area data for the readiness checks. Done when the generated table is validated against docs/difficulty.md and tested for two classes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Generated per-area table (mlvl, immunities, common elite affixes) from install data for all difficulties; validated against docs/difficulty.md section 2.1 and 3; tests for two classes
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
<!-- AC:END -->

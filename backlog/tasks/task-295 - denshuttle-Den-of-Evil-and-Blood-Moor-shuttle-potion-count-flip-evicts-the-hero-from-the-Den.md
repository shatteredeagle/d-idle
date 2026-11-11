---
id: TASK-295
title: >-
  denshuttle: Den of Evil and Blood Moor shuttle: potion-count flip evicts the
  hero from the Den
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T02:33:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: high
ordinal: 207000
card: "denshuttle"
owns:
  - "docs/denshuttle.md"
  - "internal/bot/denshuttle_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/goldfarm.go"
  - "internal/bot/broke.go"
exclusive: []
contracts: []
result: "cause V in the sandbox log (docs/denshuttle.md); fix offline only, tests for Sorceress and Paladin; NOT verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Sandbox hour 2026-11-11: 106 zone entries, 81 of them Den/Blood Moor bounces. Done when the target no longer flips out of the Den on one potion drink. Fix offline only; live proof pending.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause explained with log evidence for 5 cycles (docs/denshuttle.md)
- [x] #2 Den entry and stay potion counts have a gap, both classes tested
- [x] #3 Whole-repo gate green on the worker host
<!-- AC:END -->

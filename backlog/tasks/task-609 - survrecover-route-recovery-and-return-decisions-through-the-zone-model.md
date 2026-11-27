---
id: TASK-609
title: 'survrecover: route recovery and return decisions through the zone model'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-17T10:32:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: high
ordinal: 308000
card: "survrecover"
owns:
  - "internal/bot/zonemodel.go"
  - "internal/bot/zonerecover_test.go"
  - "internal/bot/broke.go"
  - "docs/probes/survival-ab-recovery-2026-11-17.md"
  - "internal/bot/goal.go"
  - "internal/bot/loop.go"
  - "internal/bot/observe.go"
  - "internal/bot/zoneshadow.go"
exclusive: []
contracts: []
lane: next
result: " Merged (agent/survrecover): zone-model recovery decisions and tests (zonerecover_test.go). Remaining (live): AC 3 holdout and 8-seed Paladin A/B rerun, Sorceress screen and recommendation doc."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Holdout seed 149 chickens came from the recovery state, which never reaches Plan.Pick. Done when the A/B record exists.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 recovery farm area and trip return portal ask the zone model after a cost event (knob 0 identical)
- [x] #2 tests on two classes, seed-149 sequence, no flapping
- [ ] #3 holdout and 8-seed Paladin A/B rerun, Sorceress screen, doc with recommendation
<!-- AC:END -->

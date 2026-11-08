---
id: TASK-166
title: 'pactcatalogue: Pact catalogue data file, loader, validation, incompatibility and heat rules'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:33:00Z'
labels:
  - bot
  - feature
milestone: m-4
dependencies: []
priority: low
ordinal: 142000
card: "pactcatalogue"
owns:
  - "internal/pact/catalogue.go"
  - "internal/pact/catalogue_test.go"
  - "data/pacts/pacts.json"
  - "data/pacts/pacts.go"
  - "internal/pact/export_test.go"
  - "docs/pact-catalogue.md"
exclusive: []
contracts: []
result: "Offline only. 12 Pacts (Wandering Deep and Hell left out, see docs/pact-catalogue.md), loader with named errors, heat cap and incompatibility rules tested with Paladin and Sorceress. Reward and heat numbers are U. Widened owns by the embed package data/pacts/pacts.go (embed cannot reach up from internal/)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 data/pacts/pacts.json with the 12 Pacts and the schema of pact-system.md section 4
- [x] #2 Loader errors name the cause (unknown row, knob, class, duplicate id, missing incompatibility target, rank without heat)
- [x] #3 Heat cap and incompatibility rules tested for two builds
<!-- AC:END -->

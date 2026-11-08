---
id: TASK-166
title: 'pactcatalogue: Pact catalogue data file, loader, validation, incompatibility and heat rules'
status: Proposed
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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 data/pacts/pacts.json with the 12 Pacts and the schema of pact-system.md section 4
- [ ] #2 Loader errors name the cause (unknown row, knob, class, duplicate id, missing incompatibility target, rank without heat)
- [ ] #3 Heat cap and incompatibility rules tested for two builds
<!-- AC:END -->

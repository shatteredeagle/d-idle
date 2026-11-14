---
id: TASK-413
title: 'levelsim: leveling strategy simulator and optimiser'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T11:56:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: high
ordinal: 251000
card: "levelsim"
owns:
  - "tools/levelsim/"
  - "docs/levelsim.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Simulator of Act 1 Normal leveling strategies calibrated on the four first-hour runs; finds the best ordered area strategy for 60/120/240 min and time to clvl 10/15/20.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 sim core, calibration on the logged hours, optimiser, sensitivity, CLI with tests
- [ ] #2 docs/levelsim.md and committed best-strategy summaries; plan v3 can read them
- [ ] #3 gate: gofmt, vet, tests, contracts, backlog, leakcheck
<!-- AC:END -->

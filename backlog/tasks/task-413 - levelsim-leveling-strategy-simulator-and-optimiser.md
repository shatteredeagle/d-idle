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
result: "tools/levelsim (package + CLI, tests) and docs/levelsim.md: model calibrated on the two logged hours with area data (XP at 60 min: run 4 -0.6 percent, run 1 -9.4 percent; held out -3.8 and -11.7 percent); best plan-mode strategy 20.9k XP clvl 6 at 60 min, 53.7k clvl 9 at 120, 175.5k clvl 14 at 240 against the plan's 19.5k, 39.7k, 101.7k; clvl 10 at minute 121, clvl 15 at 236. Offline, not verified live; danger floors and walking times are U."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Simulator of Act 1 Normal leveling strategies calibrated on the four first-hour runs; finds the best ordered area strategy for 60/120/240 min and time to clvl 10/15/20.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 sim core, calibration on the logged hours, optimiser, sensitivity, CLI with tests
- [x] #2 docs/levelsim.md and committed best-strategy summaries; plan v3 can read them
- [x] #3 gate: gofmt, vet, tests, contracts, backlog, leakcheck
<!-- AC:END -->

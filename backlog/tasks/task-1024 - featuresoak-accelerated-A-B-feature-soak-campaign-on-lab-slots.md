---
id: TASK-1024
title: 'featuresoak: accelerated A/B feature soak campaign on lab slots'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T21:45:00Z'
updated_date: '2026-11-27T07:01:00Z'
labels:
  - d2d
  - d2d-measure
dependencies: []
priority: medium
ordinal: 494000
card: "featuresoak"
lane: next
owns:
  - "tools/provingground/campaigns/"
  - "tools/provingground/featuresoak.py"
  - "tools/provingground/featuresoak-row.sh"
  - "docs/probes/feature-soak-2026-11-26.md"
exclusive:
  - "d2d-lab-binary"
contracts: []
result: "Lab A/B feature soak, pass 1 (1 virtual hour per row, 69 rows, 3 classes x 3 seeds, writes after start 0 in every row; lab only, not verified live): arbiter control families 3 RED (necromancer resupply/leave-game loop, 24 mode-no-exit hits), families 255 NEUTRAL, kit floor tiers NEUTRAL (sorceress gold lower), early route guard GREEN, typed outcomes NEUTRAL, reflex drink GREEN, step executor GREEN on the Cain scenarios, Paladin without Sacrifice and Necromancer defaults not exercised; lockstep speed is about 3x, 24 virtual hours per row did not run"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 campaign file, runner and row script exist and plan lists the rows
- [x] #2 stage 0 calibration table recorded
- [x] #3 per-feature ON vs OFF table with verdicts in docs/probes/feature-soak-2026-11-26.md
- [x] #4 how-to-rerun page in the doc
<!-- AC:END -->

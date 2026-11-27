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
result: "Lab A/B feature soak (lab only, not verified live). Pass 1 is mechanics-only (heroes had no class skills). Pass 2, 64 of 69 rows of 1 virtual hour, writes after start 0: no feature has a hard invariant hit ON; arbiter families 255 RED on one watchdog hang (busy loop in the hostile check), families 3 weak RED on Necromancer xp, reflex drink green on safety with more potion spend, kit floor tiers, early route guard, typed outcomes and Paladin without Sacrifice neutral, Necromancer defaults not run, step executor green on the Cain scenarios"
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

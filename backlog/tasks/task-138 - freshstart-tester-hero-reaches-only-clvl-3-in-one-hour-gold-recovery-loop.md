---
id: TASK-138
title: 'freshstart: tester hero reaches only clvl 3 in one hour (gold recovery loop)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:43:00Z'
labels:
  - bot
  - bug
  - tester
milestone: m-1
dependencies: []
priority: medium
ordinal: 114000
card: "freshstart"
owns:
  - "docs/fresh-start-pace.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by TASK-25: a fresh Sorceress in the container, realtime, goals campaign:normal_act1, stayed in the state Recovery: farming gold (gold 0 below the kit of 500) for the whole hour: 128 kills, clvl 3, gold 13, 20 to 30 gold per hour. A friend watching the first hour sees a hero that never leaves Blood Moor. Done when: the cause is named (recovery rule for a clvl 1 start with no gold, area choice, loot) and the first hour of a new character reaches the first campaign step, shown on a measured uncheated realtime run.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cause of the stall in recovery at clvl 1 named
- [ ] #2 uncheated realtime hour reaches the next campaign step
<!-- AC:END -->

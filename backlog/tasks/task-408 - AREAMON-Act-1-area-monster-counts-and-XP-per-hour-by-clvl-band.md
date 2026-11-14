---
id: TASK-408
title: 'AREAMON: Act 1 area monster counts and XP per hour by clvl band'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T11:22:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 251000
card: "areamon"
owns:
  - "docs/research/areamon-trace.md"
  - "tools/emu/am_env.sh"
  - "tools/emu/am_batch.sh"
  - "tools/emu/am_model.py"
  - "docs/research/areamon-table.csv"
exclusive: []
contracts: []
result: "Docs-only, not verified live: 12 seeds of game.exe room populations for 36 Act 1 levels under the emulator (V run), XP per clear and per minute by clvl (time model I), plan rows model reproduces hour 4 (13.1k vs 13.9k XP); Blood Moor is worst per minute, Dark Wood / Tamoe / Catacombs 2-3 best; no d2d deviation proven, no PR candidate."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: where a leveling Paladin should farm for XP per minute (hour 4: 8.2 kills/min, clvl 5). Done when: docs/research/areamon-trace.md lists 36 Act 1 levels with game.exe populations and a proposed band table. Consumer: docs/levelplan.md and the plan table owner.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Trace doc with per-area table and proposed bands
- [x] #2 Emulator runs recorded
<!-- AC:END -->

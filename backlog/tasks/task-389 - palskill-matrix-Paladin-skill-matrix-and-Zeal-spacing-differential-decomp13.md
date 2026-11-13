---
id: TASK-389
title: 'palskill-matrix: Paladin skill matrix and Zeal spacing differential (decomp13)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T23:39:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 245000
card: "palskill-matrix"
owns:
  - "docs/research/pal-skill-matrix.md"
  - "docs/research/pal-zeal-trace.md"
  - "tools/emu/zd_*"
  - "tools/emu/pm_*"
exclusive: []
contracts: []
result: "Paladin matrix (docs/research/pal-skill-matrix.md) and Zeal trace: d2d strikes Zeal 15 ticks apart (V run through the real Fight), game.exe 7 (V run, rollback FUN_00553b10): a 5-strike cast 75 vs 43 ticks, 24 of 24 gaps differ; model (I): Sacrifice 1 is x1.9 dps over Attack from clvl 2 (bot never casts it), aura ladder puts Holy Fire before Might; no PR candidate (new harness, no fix built). Offline, not verified live (2026-11-12, agent decomp13)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: TASK-330 (zealdiff), TASK-142 (paladinsmoke). Why: bot impact ranking of the Paladin skills (Sacrifice unused, Zeal d2d pace, aura ladder order). Done when: docs and tools committed, checks clean.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Paladin skill matrix doc with V/D/I tags and the skill-order answer
- [x] #2 Zeal strike spacing differential: d2d side run (TASK-330) against game.exe rollback, deviations and bot findings listed, PR candidate only if proven
<!-- AC:END -->

---
id: TASK-387
title: 'decomp14: XPGOLD trace, economy of a kill and the first-hour level gap'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-12T23:32:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 245000
card: "decomp14"
owns:
  - "docs/research/xp-gold-trace.md"
  - "tools/emu/xpg_model.py"
exclusive: []
contracts: []
result: "Offline, not verified live. XP award port equals game.exe on 119,790 of 119,790 grid lines (V run); the first-hour clvl 6 is explained by the tables (model 15,880 XP vs observed 17,032); clvl 13 needs 112,725 XP; no d2d deviation, no PR candidate. docs/research/xp-gold-trace.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Done when the trace and tools/emu/xpg_model.py are committed.

Consumer: the first-hour cards of docs/probes/hour1-analysis-2026-11-12.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc with V/S/U tags
- [x] #2 differential run and per-area table
<!-- AC:END -->

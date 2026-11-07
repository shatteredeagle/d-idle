---
id: TASK-82
title: 'fid-monster-move: oracle: monster velocity, pathing, stuck cases'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - oracle
  - d2d-monsters
  - d2d-pathing
  - d2d-movement
dependencies: []
priority: medium
ordinal: 80000
card: "fid-monster-move"
owns:
  - "docs/probes/fid/fid-monster-move.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: MA12. Method/kind: oracle. Live game (host time): no. Size: M. Fix: upstream PR (F08 monster velocity D1; path stops at 77 subtiles); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Velocity oracle (walk-speed-trace.md) re-run; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Path finding: 11 verified/6 ported/63 untouched functions: port and compare the 6 ported ones; list the rest
- [ ] #3 The 77-subtile path stop reproduced and the original behaviour read from game.exe
<!-- AC:END -->

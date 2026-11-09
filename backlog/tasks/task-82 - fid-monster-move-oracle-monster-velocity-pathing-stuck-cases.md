---
id: TASK-82
title: 'fid-monster-move: oracle: monster velocity, pathing, stuck cases'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - oracle
  - d2d-monsters
  - d2d-movement
  - d2d-pathing
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: medium
ordinal: 80000
card: "fid-monster-move"
owns:
  - "docs/probes/fid/fid-monster-move.md"
exclusive: []
contracts: []
result: "2026-11-08 decomp6: monster/player velocity equal (122,646 cases x 2 seeds, 0 mismatches), walk rates 0 of 1,203, pathers and chase check ok on 2,000 cases each, 77-point cap read (FUN_00648e70) and run; AC 2 open (63 untouched PATH functions not listed); stuck/crowd cases U; no PR candidate; doc docs/probes/fid/fid-monster-move.md; not verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: MA12. Method/kind: oracle. Live game (host time): no. Size: M. Fix: upstream PR (F08 monster velocity D1; path stops at 77 subtiles); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Velocity oracle (walk-speed-trace.md) re-run; tolerance: 0 differing cases in the oracle run; a deviation...; Path finding: 11 verified/6 ported/63 untouched functions: port and compare the 6 ported ones; list the rest; The 77-subtile path stop reproduced and the original behaviour read from game.exe (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Velocity oracle (walk-speed-trace.md) re-run; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Path finding: 11 verified/6 ported/63 untouched functions: port and compare the 6 ported ones; list the rest
- [x] #3 The 77-subtile path stop reproduced and the original behaviour read from game.exe
<!-- AC:END -->

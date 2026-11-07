---
id: TASK-79
title: 'fid-inv-limits: oracle: stash cap, purse cap, grid sizes'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - oracle
  - d2d-items
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: low
ordinal: 77000
card: "fid-inv-limits"
owns:
  - "docs/probes/fid/fid-inv-limits.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: IT10. Method/kind: oracle. Live game (host time): no. Size: S. Fix: none expected; upstream PR only if the purse cap differs. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Trace the purse cap clvl x 10,000 in game.exe (FUN_00623460 neighbourhood); expected equal to the d2d clamp; Grid sizes (inventory 10x4, stash 6x8, belt rows) from the data tables of both sides; expected equal,... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Trace the purse cap clvl x 10,000 in game.exe (FUN_00623460 neighbourhood); expected equal to the d2d clamp
- [ ] #2 Grid sizes (inventory 10x4, stash 6x8, belt rows) from the data tables of both sides; expected equal, tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
<!-- AC:END -->

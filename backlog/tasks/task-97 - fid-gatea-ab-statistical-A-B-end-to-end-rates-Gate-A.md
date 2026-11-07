---
id: TASK-97
title: 'fid-gatea-ab: statistical A:B: end-to-end rates, Gate A'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-07T18:04:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - d2d-measure
  - d2d-combat
  - d2d-monsters
  - d2d-time
  - fidelity
  - gate
  - soak
  - parked
milestone: m-2
dependencies:
  - TASK-55
  - TASK-56
  - TASK-72
  - TASK-92
priority: high
ordinal: 95000
card: "fid-gatea-ab"
owns:
  - "docs/probes/fid/fid-gatea-ab.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07; Gate A and Gate 3 runs wait for the design brief v1 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: EN03. Method/kind: d2d-measure. Live game (host time): yes (6 game.exe + 6 d2d lanes, soak slots, about 40 h). Size: L. Fix: bot or upstream depending on the gap; the arbiter. Dependencies: fid-lab-rebase; fid-probe-readers; slices 1 to 3. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Fixed seed set run on 6 game.exe and 6 d2d lanes at fresh clvl 1 to 6; tolerance:...; XP/h, kills/h, deaths/h table with paired interval in docs/probes/fid/fid-gatea-ab.md; Every lane record carries writes after start: 0 (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fixed seed set run on 6 game.exe and 6 d2d lanes at fresh clvl 1 to 6; tolerance: docs/contracts/fidelity-gate.md gap rule (20 %)
- [ ] #2 XP/h, kills/h, deaths/h table with paired interval in docs/probes/fid/fid-gatea-ab.md
- [ ] #3 Every lane record carries writes after start: 0
<!-- AC:END -->

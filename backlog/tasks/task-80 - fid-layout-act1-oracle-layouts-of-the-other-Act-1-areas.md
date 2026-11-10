---
id: TASK-80
title: 'fid-layout-act1: oracle: layouts of the other Act 1 areas'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - oracle
  - d2d-levels
  - d2d-pathing
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: high
ordinal: 78000
card: "fid-layout-act1"
owns:
  - "docs/probes/fid/fid-layout-act1.md"
exclusive: []
contracts: []
result: "decomp8 2026-11-09, offline only (not verified live): ids 5-7 and 9-39 equal game.exe on upstream 83fc7ba1, tiles/units/seeds/monsters, seeds 1..30 and two 32-bit ranges, 0 differ (170 + 340 comparisons, 13,497 + 9,184 populated rooms); Lut Gholein now equal; AC 3 (same-save first load) open, needs a game; docs/probes/fid/fid-layout-act1.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: MA04. Method/kind: oracle. Live game (host time): no (offline emulator, tools/emu seedspawn). Size: L. Fix: upstream PR where the DRLG differs (0 of 102 outdoor ports verified); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Extend the seedspawn oracle to Burial Grounds, Cave 1/2, Dark Wood, Black Marsh, Tamoe Highland, Outer...; Per-area result table in docs/probes/fid/fid-layout-act1.md (cases, differ, fix state); Same-save check (scenario 8, Stony Field) first-load monster/object list equals on a pinned seed (and 1 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Extend the seedspawn oracle to Burial Grounds, Cave 1/2, Dark Wood, Black Marsh, Tamoe Highland, Outer Cloister, Barracks/Jail, Catacombs 1-4, Forgotten Tower, Crypt, Mausoleum, Tristram; map seeds 1..30; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [x] #2 Per-area result table in docs/probes/fid/fid-layout-act1.md (cases, differ, fix state)
- [ ] #3 Same-save check (scenario 8, Stony Field) first-load monster/object list equals on a pinned seed
- [x] #4 Lut Gholein tile mismatch (act 2) recorded as an open row, not fixed here
<!-- AC:END -->

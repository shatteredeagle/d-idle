---
id: TASK-80
title: 'fid-layout-act1: oracle: layouts of the other Act 1 areas'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
labels:
  - d2d
  - oracle
  - d2d-levels
  - d2d-pathing
  - fidelity
milestone: m-2
dependencies: []
priority: high
ordinal: 78000
card: "fid-layout-act1"
owns:
  - "docs/probes/fid/fid-layout-act1.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: MA04. Method/kind: oracle. Live game (host time): no (offline emulator, tools/emu seedspawn). Size: L. Fix: upstream PR where the DRLG differs (0 of 102 outdoor ports verified); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Extend the seedspawn oracle to Burial Grounds, Cave 1/2, Dark Wood, Black Marsh, Tamoe Highland, Outer...; Per-area result table in docs/probes/fid/fid-layout-act1.md (cases, differ, fix state); Same-save check (scenario 8, Stony Field) first-load monster/object list equals on a pinned seed (and 1 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Extend the seedspawn oracle to Burial Grounds, Cave 1/2, Dark Wood, Black Marsh, Tamoe Highland, Outer Cloister, Barracks/Jail, Catacombs 1-4, Forgotten Tower, Crypt, Mausoleum, Tristram; map seeds 1..30; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Per-area result table in docs/probes/fid/fid-layout-act1.md (cases, differ, fix state)
- [ ] #3 Same-save check (scenario 8, Stony Field) first-load monster/object list equals on a pinned seed
- [ ] #4 Lut Gholein tile mismatch (act 2) recorded as an open row, not fixed here
<!-- AC:END -->

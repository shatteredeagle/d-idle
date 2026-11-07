---
id: TASK-75
title: 'fid-xp-award: oracle plus live: XP per kill and level-difference penalty'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - readonly
  - d2d-monsters
  - d2d-combat
dependencies: []
priority: medium
ordinal: 73000
card: "fid-xp-award"
owns:
  - "docs/probes/fid/fid-xp-award.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: ST10. Method/kind: readonly. Live game (host time): yes, one short kill count on game.exe (about 20 min) after the oracle. Size: S. Fix: none expected (Den +60 % was composition); upstream fix 01337ba3 exists. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Re-run the xp-award grid (119,790 lines) against current main; expected 0 differ (tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR)
- [ ] #2 Live read of XP before/after a Fallen, Zombie and Dark Hunter kill at clvl 4 to 6 on game.exe (read-only); expected 18/33/known values of docs/research/xp-award-trace.md exactly
<!-- AC:END -->

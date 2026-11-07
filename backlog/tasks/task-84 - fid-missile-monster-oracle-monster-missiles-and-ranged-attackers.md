---
id: TASK-84
title: 'fid-missile-monster: oracle: monster missiles and ranged attackers'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - oracle
  - d2d-monsters
  - d2d-combat
dependencies: []
priority: high
ordinal: 82000
card: "fid-missile-monster"
owns:
  - "docs/probes/fid/fid-missile-monster.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: MA10. Method/kind: oracle. Live game (host time): no. Size: M. Fix: upstream PR (F14 missile block gate, MissA to-hit); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Run the 371 untested of 684 missile rows needed by Act 1 ranged monsters (archers, Blood Raven arrows, Fetish dart, lightning enchanted); tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 F14 block gate and MissA to-hit differential with a proof table
- [ ] #3 Soaky death has no d2d twin: decide documented or ported, in docs/probes/fid/fid-missile-monster.md
<!-- AC:END -->

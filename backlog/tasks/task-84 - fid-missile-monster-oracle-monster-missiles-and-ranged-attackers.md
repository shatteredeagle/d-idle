---
id: TASK-84
title: 'fid-missile-monster: oracle: monster missiles and ranged attackers'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - d2d-monsters
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: high
ordinal: 82000
card: "fid-missile-monster"
owns:
  - "docs/probes/fid/fid-missile-monster.md"
exclusive: []
contracts: []
result: "Partial: launch record 60,000 cases x 36 rows 0 differ on real build (V run); to-hit gate merged upstream; block gate still open on d2d (V read); 371-row count not closed. AC1 not ticked. Doc docs/probes/fid/fid-missile-monster.md. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: MA10. Method/kind: oracle. Live game (host time): no. Size: M. Fix: upstream PR (F14 missile block gate, MissA to-hit); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Run the 371 untested of 684 missile rows needed by Act 1 ranged monsters (archers, Blood Raven arrows,...; F14 block gate and MissA to-hit differential with a proof table; Soaky death has no d2d twin: decide documented or ported, in docs/probes/fid/fid-missile-monster.md (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Run the 371 untested of 684 missile rows needed by Act 1 ranged monsters (archers, Blood Raven arrows, Fetish dart, lightning enchanted); tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 F14 block gate and MissA to-hit differential with a proof table
- [ ] #3 Soaky death has no d2d twin: decide documented or ported, in docs/probes/fid/fid-missile-monster.md
<!-- AC:END -->

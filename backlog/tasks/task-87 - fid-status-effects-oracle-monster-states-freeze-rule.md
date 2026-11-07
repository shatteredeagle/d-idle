---
id: TASK-87
title: 'fid-status-effects: oracle: monster states, freeze rule'
status: Proposed
assignee: []
created_date: '2026-11-07T18:04:00Z'
labels:
  - fidelity
  - oracle
  - d2d-monsters
  - d2d-combat
  - d2d-skills
dependencies: []
priority: medium
ordinal: 85000
card: "fid-status-effects"
owns:
  - "docs/probes/fid/fid-status-effects.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: MA11 SK08. Method/kind: oracle. Live game (host time): no. Size: M. Fix: upstream PR (freeze rule C3 to C5, 22 state deviations); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 states: 149 untested rows; run the ones used by freeze/chill, curse, poison, stun; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Freeze rule 225,748 cases deviations C3 to C5 reduced by the proof PR (pr-freeze-proof.md)
- [ ] #3 Ice Bolt chill vs cold-immune quirk checked
<!-- AC:END -->

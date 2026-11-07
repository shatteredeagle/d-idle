---
id: TASK-90
title: 'fid-teleport: oracle: Teleport hops, refusal and mana'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T18:04:00Z'
labels:
  - d2d
  - oracle
  - d2d-movement
  - d2d-skills
  - fidelity
milestone: m-2
dependencies:
  - TASK-92
priority: medium
ordinal: 88000
card: "fid-teleport"
owns:
  - "docs/probes/fid/fid-teleport.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: SK11. Method/kind: oracle. Live game (host time): no (a live cast is optional; never reached in a measured run). Size: M. Fix: upstream PR (batch 2 D1 room, D2 blocked target; private, not open upstream); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: lab-binary rebase. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: D3/D4 mana on a refused hop compared; tolerance: 0 differing cases in the oracle run; a deviation becomes...; D1/D2 batch 2 proof table finalised; Live: one Teleport to a blocked target on a clvl 18 character reads mana unchanged, if the maintainer wants it (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 D3/D4 mana on a refused hop compared; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 D1/D2 batch 2 proof table finalised
- [ ] #3 Live: one Teleport to a blocked target on a clvl 18 character reads mana unchanged, if the maintainer wants it
<!-- AC:END -->

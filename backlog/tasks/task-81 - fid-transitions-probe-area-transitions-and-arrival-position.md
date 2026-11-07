---
id: TASK-81
title: 'fid-transitions: probe: area transitions and arrival position'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - probe
  - d2d-levels
  - d2d-movement
dependencies: []
priority: medium
ordinal: 79000
card: "fid-transitions"
owns:
  - "docs/probes/fid/fid-transitions.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: PR09. Method/kind: probe. Live game (host time): yes (host time about 30 min, input for walking through a warp). Size: S. Fix: bot shim R05 until a d2d fix; upstream PR only if arrival differs. Dependencies: TASK-55. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Take the Blood Moor to Den cave mouth and Cold Plains border on both sides from a save; expected same arrival subtile within 1 cell
- [ ] #2 Arrival lies inside or outside the trigger as on game.exe (shim R05 re-checked); tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
<!-- AC:END -->

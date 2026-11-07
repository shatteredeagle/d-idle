---
id: TASK-81
title: 'fid-transitions: probe: area transitions and arrival position'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - probe
  - d2d-levels
  - d2d-movement
  - fidelity
  - parked
milestone: m-2
dependencies:
  - TASK-55
priority: medium
ordinal: 79000
card: "fid-transitions"
owns:
  - "docs/probes/fid/fid-transitions.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: PR09. Method/kind: probe. Live game (host time): yes (host time about 30 min, input for walking through a warp). Size: S. Fix: bot shim R05 until a d2d fix; upstream PR only if arrival differs. Dependencies: TASK-55. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Take the Blood Moor to Den cave mouth and Cold Plains border on both sides from a save; expected same...; Arrival lies inside or outside the trigger as on game.exe (shim R05 re-checked); tolerance from... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Take the Blood Moor to Den cave mouth and Cold Plains border on both sides from a save; expected same arrival subtile within 1 cell
- [ ] #2 Arrival lies inside or outside the trigger as on game.exe (shim R05 re-checked); tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
<!-- AC:END -->

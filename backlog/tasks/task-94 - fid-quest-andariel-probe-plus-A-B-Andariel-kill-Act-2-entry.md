---
id: TASK-94
title: 'fid-quest-andariel: probe plus A:B: Andariel kill, Act 2 entry'
status: Proposed
assignee: []
created_date: '2026-11-07T18:04:00Z'
labels:
  - fidelity
  - drill
  - d2d-levels
  - d2d-monsters
  - gate
dependencies: []
priority: medium
ordinal: 92000
card: "fid-quest-andariel"
owns:
  - "docs/probes/fid/fid-quest-andariel.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: PR04. Method/kind: drill. Live game (host time): yes (kill is a soak slot; Gate A bar). Size: L. Fix: bot change if the quest FSM differs; part of the Gate A bar (decisions.md Current truth). Dependencies: TASK-55; fid-boss-skills. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Save with Andariel dead, in Lut Gholein: difficulty unlock, Warriv act change on both sides; tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
- [ ] #2 Act 2 town layout and waypoint list (scenario 12), Lut Gholein tile mismatch recorded
- [ ] #3 Statistical check only within tolerance: docs/contracts/fidelity-gate.md gap rule (20 %)
<!-- AC:END -->

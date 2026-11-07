---
id: TASK-94
title: 'fid-quest-andariel: probe plus A:B: Andariel kill, Act 2 entry'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-07T18:04:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - bot
  - drill
  - d2d-levels
  - d2d-monsters
  - fidelity
  - gate
  - parked
milestone: m-2
dependencies:
  - TASK-55
  - TASK-86
priority: medium
ordinal: 92000
card: "fid-quest-andariel"
owns:
  - "docs/probes/fid/fid-quest-andariel.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: PR04. Method/kind: drill. Live game (host time): yes (kill is a soak slot; Gate A bar). Size: L. Fix: bot change if the quest FSM differs; part of the Gate A bar (decisions.md Current truth). Dependencies: TASK-55; fid-boss-skills. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Save with Andariel dead, in Lut Gholein: difficulty unlock, Warriv act change on both sides; tolerance...; Act 2 town layout and waypoint list (scenario 12), Lut Gholein tile mismatch recorded; Statistical check only within tolerance: docs/contracts/fidelity-gate.md gap rule (20 %) (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Save with Andariel dead, in Lut Gholein: difficulty unlock, Warriv act change on both sides; tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
- [ ] #2 Act 2 town layout and waypoint list (scenario 12), Lut Gholein tile mismatch recorded
- [ ] #3 Statistical check only within tolerance: docs/contracts/fidelity-gate.md gap rule (20 %)
<!-- AC:END -->

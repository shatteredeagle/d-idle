---
id: TASK-93
title: 'fid-quest-den-raven: probe: Den of Evil and Blood Raven triggers'
status: Proposed
assignee: []
created_date: '2026-11-07T18:04:00Z'
labels:
  - fidelity
  - probe
  - d2d-monsters
  - d2d-town
  - d2d-levels
dependencies: []
priority: high
ordinal: 91000
card: "fid-quest-den-raven"
owns:
  - "docs/probes/fid/fid-quest-den-raven.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: PR01 PR02. Method/kind: probe. Live game (host time): yes (scenario 11 plus two saves with quest words set). Size: M. Fix: upstream PR if a trigger or reward differs; bot change for the claim flow; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: TASK-55 (scenario 11). Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Save before the last Den monster: kill it on both sides; expected same quest word and Akara skill point reward; tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
- [ ] #2 Save after Blood Raven: Kashya claim flow reaches the free Rogue on both sides (quest FSM, hireling.txt); expected same merc class and level
- [ ] #3 Quest/waypoint words (PR05) file == game.exe == d2d
<!-- AC:END -->

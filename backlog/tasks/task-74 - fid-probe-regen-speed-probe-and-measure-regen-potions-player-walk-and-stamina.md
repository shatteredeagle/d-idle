---
id: TASK-74
title: 'fid-probe-regen-speed: probe and measure: regen, potions, player walk and stamina'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - d2d-measure
  - d2d-combat
  - d2d-movement
  - d2d-time
dependencies: []
priority: medium
ordinal: 72000
card: "fid-probe-regen-speed"
owns:
  - "docs/probes/fid/fid-probe-regen-speed.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: ST04 ST05 ST08. Method/kind: d2d-measure. Live game (host time): yes (game.exe side needs input for potions and walking; open question 4 of the matrix). Size: M. Fix: upstream PR if a deviation is proven (potion double fill missing 0/24); else bot shim in docs/d2d-deviations.md. Dependencies: TASK-55; user answer to matrix open question 4. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Natural mana regen: two reads N s apart in town on both sides; expected floor formula equal, tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
- [ ] #2 Potion: Minor Healing 30 life over 7.68 s, 24 drinks, and the double fill by Vit on game.exe; expected d2d equals (d2d currently misses double fill 0/24)
- [ ] #3 Chain drinking run on both sides; difference listed
- [ ] #4 Walk speed (cells/s) and stamina drain/regen per tick on game.exe vs d2d 1.87 cells/s and max/256 per tick; tolerance 0.5 %
<!-- AC:END -->

---
id: TASK-72
title: 'fid-probe-readers: probe reader gaps: d2s third reference, merc fields, affix values'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - probe
  - d2d-items
  - d2d-data
  - tooling
dependencies: []
priority: medium
ordinal: 70000
card: "fid-probe-readers"
owns:
  - "docs/probes/fid/fid-probe-readers.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: ST07 IT04 ME06. Method/kind: probe. Live game (host time): no (offline, then covered by the TASK-55 live run). Size: M. Fix: bot change (tooling in internal/fidprobe). Dependencies: TASK-55. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Third reference: saves.Decode of the .d2s fills the Sheet; scenario 1 replay shows file == game.exe == d2d for ST07 base stats (Str 10 Dex 25 Vit 10 Ene 35 for the Sorceress), tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
- [ ] #2 Merc fields (class, level, life, resists, skills) readable on both sides; scenario 6 replay lists no UNREADABLE merc group (ME06)
- [ ] #3 Item affix values (IT04) read on both sides for scenario 10 fixture; UNREADABLE share below 5 % (probe design section 6)
- [ ] #4 d2d leveler 'unreadable' path (d2d-wire.md, ST07) reads base stats through the d2d source; unit test on a recorded dump
<!-- AC:END -->

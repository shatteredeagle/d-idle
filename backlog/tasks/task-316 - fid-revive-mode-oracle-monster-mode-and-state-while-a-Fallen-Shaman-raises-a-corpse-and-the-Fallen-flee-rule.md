---
id: TASK-316
title: >-
  fid-revive-mode: oracle: monster mode and state while a Fallen Shaman raises a
  corpse, and the Fallen flee rule
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T18:50:00Z'
updated_date: '2026-11-11T21:25:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - fidelity
dependencies: []
priority: medium
ordinal: 217000
card: "fid-revive-mode"
owns:
  - "docs/probes/fid/fid-revive-mode.md"
exclusive: []
contracts: []
result: "Raised monster is alive at once: life = max, mode DD 12 -> MonStats2 ResurrectMode (Fallen and Mummy S1 8, skeleton families SQ 14, rest NU 1; 703 spawnable rows run natively, 578 compared with the column, 0 disagree); d2d sets NU for all (D1, game side V run, d2d side V read; no PR candidate). Flee rule restated from monsterai2 (dying kin within 15, pace +50; no low-life flee, I). Offline (2026-11-11, agent decomp9)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: TASK-142 (paladinsmoke): the bot's Alive() and hostile filters in internal/d2/units.go. Fidelity rows MA08 MA13.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc written with V run / V read / D / U tags
- [x] #2 deviations and bot findings listed, no PR candidate unless the differential proves it
<!-- AC:END -->

---
id: TASK-317
title: >-
  fid-repair-wear: oracle: repair cost formula and durability loss rates, sizing
  for the repair trip
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
ordinal: 218000
card: "fid-repair-wear"
owns:
  - "docs/probes/fid/fid-repair-wear.md"
exclusive: []
contracts: []
result: "Repair formula and wear rates joined from the existing V run proofs (repair = cost x missing / max / 8 at every smith; weapon 4 percent and armour 10 percent per landed melee blow, split 5/3/4/4/2/2/2); sizing table for the trip (starting gear: all repairs under 60 gold, first piece at 20 percent after 300+ blows received); nothing new deviates, bot RepairPrice equals the proof. Offline, no new run (2026-11-11, agent decomp9)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: TASK-226 (repairmanual): trip sizing and the repair estimate. Fidelity row IT08.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc written with V run / V read / D / U tags
- [x] #2 deviations and bot findings listed, no PR candidate unless the differential proves it
<!-- AC:END -->

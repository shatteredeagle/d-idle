---
id: TASK-318
title: 'fid-pal-attack: oracle: Paladin Attack and Zeal speed, mana and profile bands'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T18:51:00Z'
updated_date: '2026-11-11T21:25:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - fidelity
dependencies: []
priority: medium
ordinal: 219000
card: "fid-pal-attack"
owns:
  - "docs/probes/fid/fid-pal-attack.md"
exclusive: []
contracts: []
result: "Paladin Attack ticks tabulated (15 for a 1H sword at IAS 0; 13 Scimitar), mana per skill from the SKMANA formula (Zeal 2 per cast), bot bands equal the skill req levels; Zeal strike interval NOT resolved (U; FUN_00553b10 harness dropped; d2d waits a full swing per strike, D9 of the Smite/Zeal trace). No PR candidate. Offline (2026-11-11, agent decomp9)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: TASK-142 (paladinsmoke): the paladin bands in internal/combat/classdata/profiles.json. Fidelity rows SK04 SK16 plus the Paladin skill rows.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc written with V run / V read / D / U tags
- [x] #2 deviations and bot findings listed, no PR candidate unless the differential proves it
<!-- AC:END -->

---
id: TASK-92
title: 'fid-lab-rebase: lab binary rebase: private fixes into the lab binary'
status: Proposed
assignee: []
created_date: '2026-11-07T18:04:00Z'
labels:
  - fidelity
  - d2d-measure
  - d2d-data
  - d2d-time
dependencies: []
priority: high
ordinal: 90000
card: "fid-lab-rebase"
owns:
  - "docs/probes/fid/fid-lab-rebase.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: SK05 SK06 SK07 SK11 IT09 (all private-branch DEV rows). Method/kind: d2d-measure. Live game (host time): yes (a lab binary smoke run, host time about 1 h). Size: M. Fix: fork work, no bot change; commit and push to the private fork the same day. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 List every private-branch fix (R01 to R06, group stacking, batch 2) with its branch and commit on the private remote
- [ ] #2 Rebuilt lab binary: oracle spot checks of SK05 and SK06 show 0 differ against game.exe
- [ ] #3 Smoke: one probe scenario (TASK-55) runs against the rebased binary and the old one; delta listed
<!-- AC:END -->

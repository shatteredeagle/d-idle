---
id: TASK-76
title: 'fid-price-pr: oracle: buy, sell and repair price deviations, gold plans'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - upstream-pr
  - d2d-vendor
  - d2d-items
  - d2d-data
dependencies: []
priority: high
ordinal: 74000
card: "fid-price-pr"
owns:
  - "docs/probes/fid/fid-price-pr.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: VN01 VN02 VN03 IT08 VN06. Method/kind: upstream-pr. Live game (host time): no (offline; the live quote check is in TASK-56). Size: M. Fix: upstream d2d PR (price-trace.md Q1,Q2,Q5, pr-vendor-repair-gate-draft.md); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none (live confirmation by TASK-56). Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Price oracle re-run on current main: buy differs 39,626/100,000, sell 38,939/100,000, repair 42,705/100,000 before the fix; expected 0 differ after; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 VN03 consumable prices: TP scroll 100, ID 80, Minor HP 30, Minor MP 60, key 45, tomes 250/200 vs game.exe table; the live 450 tome price explained or closed
- [ ] #3 PR draft with the differential proof in docs/probes/fid/fid-price-pr.md; send only on the user's go
- [ ] #4 VN06: re-calibrate the early gold plan (sell 50 %) and show the plan numbers unchanged or listed (bot side)
<!-- AC:END -->

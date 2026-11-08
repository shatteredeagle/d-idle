---
id: TASK-76
title: 'fid-price-pr: oracle: buy, sell and repair price deviations, gold plans'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-08T00:29:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-data
  - d2d-items
  - d2d-vendor
  - fidelity
milestone: m-2
dependencies:
  - TASK-56
priority: high
ordinal: 74000
card: "fid-price-pr"
owns:
  - "docs/probes/fid/fid-price-pr.md"
  - "docs/research/fid-vendor-price-trace.md"
  - "tools/emu/fp_clamp.py"
  - "tools/emu/fp_price_runs.sh"
  - "docs/research/pr-price2-draft.md"
exclusive: []
contracts: []
result: "ALREADY UPSTREAM, no PR candidate (2026-11-08, agent decomp5 re-run on upstream 1453f853): seed 7 buy 1 / sell 0 / repair 13 of 100,000 (the 2^31 overflow family at 26M+ prices, I unreachable), seed 1 0/0/16, tomes 0 of 144; docs/probes/fid/fid-price-pr.md. AC 1, 3, 4 stay open (before-numbers outdated, nothing to send, bot gold plan not touched). Offline emulator runs only. Status sync (stays open): AC #1, #3, #4 unticked, nothing to send upstream."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: VN01 VN02 VN03 IT08 VN06. Method/kind: upstream-pr. Live game (host time): no (offline; the live quote check is in TASK-56). Size: M. Fix: upstream d2d PR (price-trace.md Q1,Q2,Q5, pr-vendor-repair-gate-draft.md); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none (live confirmation by TASK-56). Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Price oracle re-run on current main: buy differs 39,626/100,000, sell 38,939/100,000, repair...; VN03 consumable prices: TP scroll 100, ID 80, Minor HP 30, Minor MP 60, key 45, tomes 250/200 vs game.exe...; PR draft with the differential proof in docs/probes/fid/fid-price-pr.md; send only on the user's go (and 1 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Price oracle re-run on current main: buy differs 39,626/100,000, sell 38,939/100,000, repair 42,705/100,000 before the fix; expected 0 differ after; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [x] #2 VN03 consumable prices: TP scroll 100, ID 80, Minor HP 30, Minor MP 60, key 45, tomes 250/200 vs game.exe table; the live 450 tome price explained or closed
- [ ] #3 PR draft with the differential proof in docs/probes/fid/fid-price-pr.md; send only on the user's go
- [ ] #4 VN06: re-calibrate the early gold plan (sell 50 %) and show the plan numbers unchanged or listed (bot side)
<!-- AC:END -->

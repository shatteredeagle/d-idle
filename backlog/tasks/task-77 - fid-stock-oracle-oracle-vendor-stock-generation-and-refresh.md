---
id: TASK-77
title: 'fid-stock-oracle: oracle: vendor stock generation and refresh'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - oracle
  - d2d-vendor
  - d2d-items
  - d2d-loot
dependencies: []
priority: high
ordinal: 75000
card: "fid-stock-oracle"
owns:
  - "docs/probes/fid/fid-stock-oracle.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: VN04 VN05. Method/kind: oracle. Live game (host time): yes only for the TASK-56 stock read; the oracle itself is offline. Size: L. Fix: upstream PR if stock generation differs; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: TASK-56 (stock lists), probe core. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decompile and document the NPC stock generation (ilvl rule, item count, magic affixes) for Akara, Charsi, Gheed, Kashya in docs/probes/fid/fid-stock-oracle.md
- [ ] #2 Oracle differential of stock generation d2d vs game.exe on 30 seeds; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #3 Stock refresh rule (new game, level-up, time) read from game.exe and compared with d2d; the SME 'low confidence' note is closed
- [ ] #4 TASK-56 live stock lists agree with the oracle or the difference is listed per vendor
<!-- AC:END -->

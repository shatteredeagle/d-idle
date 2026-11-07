---
id: TASK-98
title: 'fid-adapter-rows: model-check: adapter and policy rows'
status: Deferred
assignee: []
created_date: '2026-11-07T18:04:00Z'
labels:
  - fidelity
  - model-check
  - d2d-ui
  - d2d-town
  - d2d-time
  - parked
dependencies: []
priority: low
ordinal: 96000
card: "fid-adapter-rows"
owns:
  - "docs/probes/fid/fid-adapter-rows.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: SK14 IT06 VN08 VN09 ME06 EN01. Method/kind: model-check. Live game (host time): no. Size: S. Fix: none; adapter or bot policy, not a fidelity question. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Per row, re-read the matrix note against the code once (hotkey bind, pickit policy, vendor tab compaction, no gamble code, hire/claim reader, env clock) and confirm it still holds
- [ ] #2 Re-open only if an adapter changes (parked: no fidelity question)
<!-- AC:END -->

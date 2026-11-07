---
id: TASK-103
title: 'fid-poison-res-diag: diagnose Freshe poison resist 6 in d2d, absent in game.exe'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:49:00Z'
labels:
  - d2d
  - oracle
  - d2d-items
  - fidelity
milestone: m-2
dependencies:
  - TASK-72
priority: low
ordinal: 101000
card: "fid-poison-res-diag"
owns:
  - "docs/research/fid-poisonres-trace.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Matrix rows IT04 ST06. Evidence: docs/research/fidelity-probe-run1.md finding 3 (Freshe stat 45 = 6 in d2d, none in game.exe, scenarios 3 and 4; the Paladin has no difference). Hypothesis (S): a socketed gem in the helm. Needs the item reader first (TASK-72).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 d2d item list with socket contents readable (TASK-72) and the source of stat 45 named
- [ ] #2 game.exe side read for the same item (rule or reader scope)
- [ ] #3 Verdict: d2d bug, game.exe reader scope, or equal; matrix row updated
<!-- AC:END -->

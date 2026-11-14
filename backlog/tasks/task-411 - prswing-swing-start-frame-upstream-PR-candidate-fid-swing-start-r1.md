---
id: TASK-411
title: 'prswing: swing start frame upstream PR candidate (fid/swing-start-r1)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-14T11:32:00Z'
updated_date: '2026-11-14T13:45:00Z'
labels:
  - d2d
  - upstream-pr
dependencies: []
priority: medium
ordinal: 251000
card: "prswing"
owns:
  - "docs/upstream/drafts/swing-start.md"
  - "docs/upstream/pr-queue.md"
exclusive:
  - "fork-branch:fid/swing-start-r1"
  - "fork-branch:fork-only/sandbox-swing"
contracts: []
result: "fid/swing-start-r1 (5af717cd oracle, ebeb4268 claim, on upstream f9b8cfed), stacked fork-only/sandbox-swing 20a10eda, both pushed to the private fork; oracle 12,460 cells 2,102 differ before, 0 after; Zeal grid 200 of 600 before, 0 after; ctest only the 4 known failures; presubmit 0 FAIL; kill-rate A/B (Sorceress and Paladin, seeds 7 and 11, 10 min, one run a cell): no measurable change, noise exceeds any effect (worker host, V run); draft ready for the independent verifier pass; nothing posted"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: decomp15's SWING1 trace (TASK-403) found game.exe's swing start frame (Amazon, Sorceress) and strike tick closed form that d2d lacks. Build it as an upstream PR candidate on the private fork (fid/swing-start-r1, two commits oracle then claim on upstream main f9b8cfed), stack it fork-only (fork-only/sandbox-swing), draft the PR body, queue row 1c. Done when: the differential is 0 after, the draft says ready for the independent verifier pass. Nothing is posted upstream.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Branch fid/swing-start-r1 on upstream main f9b8cfed: oracle commit then fix+test commit, pushed to the private fork
- [x] #2 Differential: 12,460 cells, 2,102 differ before, 0 after
- [x] #3 Draft docs/upstream/drafts/swing-start.md and pr-queue row: ready for independent verifier pass
<!-- AC:END -->

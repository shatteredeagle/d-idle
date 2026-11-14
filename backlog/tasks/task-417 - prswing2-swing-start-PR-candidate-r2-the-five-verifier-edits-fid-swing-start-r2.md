---
id: TASK-417
title: >-
  prswing2: swing start PR candidate r2, the five verifier edits
  (fid/swing-start-r2)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-14T15:00:00Z'
updated_date: '2026-11-14T15:07:00Z'
labels:
  - d2d
  - upstream-pr
dependencies: []
priority: medium
ordinal: 253000
card: "prswing2"
owns:
  - "docs/upstream/drafts/swing-start.md"
  - "docs/upstream/pr-queue.md"
exclusive:
  - "fork-branch:fid/swing-start-r2"
  - "fork-branch:fork-only/sandbox-swing"
contracts: []
result: "fid/swing-start-r2 (4a18fa9e oracle, 8626e244 claim, on upstream f9b8cfed) pushed to the private fork; held weapon = the usable one; oracle 11,270 cells 2,011 differ before, 0 after; verifier's 19,800-cell probe 0 of 19,170 after (630 class-restricted-gear cells unreachable); ctest only the 4 known failures; presubmit 0 FAIL, pr_guard 0; fork-only/sandbox-swing carries r2 (2e58bdc6), image not swapped; nothing posted"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the first verifier pass (TASK-415) found NEEDS-WORK on fid/swing-start-r1. Build r2 with the five edits, re-run the grids, update the draft and queue row 1c. Done when: the second verifier pass can start. Nothing is posted.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 fid/swing-start-r2 on upstream f9b8cfed with the five edits of TASK-415, pushed to the private fork
- [x] #2 Oracle 11,270 cells and the verifier's 19,800-cell grid: 0 differ after (class-restricted gear excluded)
- [x] #3 Draft and queue row 1c updated: ready for the second verifier pass
<!-- AC:END -->

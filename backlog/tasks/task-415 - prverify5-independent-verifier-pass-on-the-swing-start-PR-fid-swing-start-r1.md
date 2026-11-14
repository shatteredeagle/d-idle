---
id: TASK-415
title: >-
  prverify5: independent verifier pass on the swing-start PR
  (fid/swing-start-r1)
status: Done
assignee:
  - live-agent
created_date: '2026-11-14T13:51:00Z'
updated_date: '2026-11-14T14:24:00Z'
labels:
  - d2d
  - research
dependencies: []
priority: medium
ordinal: 252000
card: "prverify5"
owns:
  - "docs/upstream/verification-swing-start-2026-11-14.md"
exclusive: []
contracts: []
result: "NEEDS-WORK on r1 (not verified live; offline differential V run on the worker host): rule re-derived and 0 of 19,260 new real-path cells differ after; required: held weapon must skip broken / requirement-short items (505 cells), combat.md count 1,195/11,200 -> 1,267/11,620, Fend is A1 not a sequence, throws get the strike rule; docs/upstream/verification-swing-start-2026-11-14.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user's rule: an independent verifier must pass before any upstream PR is posted (TASK-411 prswing, queue row 1c). Consumer: TASK-411. Done when: the verdict doc is committed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Verdict PASS / NEEDS-WORK / FAIL with exact edits in docs/upstream/verification-swing-start-2026-11-14.md
- [x] #2 Rule re-derived from the disassembly and differential re-run on new grids, parent vs claim
<!-- AC:END -->

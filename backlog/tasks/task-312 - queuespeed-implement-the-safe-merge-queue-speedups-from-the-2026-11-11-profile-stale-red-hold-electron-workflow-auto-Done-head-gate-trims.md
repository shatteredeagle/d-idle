---
id: TASK-312
title: 'queuespeed: implement the safe merge-queue speedups from the 2026-11-11 profile (stale red hold, electron workflow, auto-Done head, gate trims)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T17:14:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 215000
card: "queuespeed"
owns:
  - "docs/process/queuespeed.md"
exclusive: []
contracts: []
result: "Steps 1 to 3 implemented in the ops repo (tools/mergeq.py, branch queuespeed of the ops repo; tests in tools/tests/test_mergeq.py against a scratch bare repo with fake runs), MERGE_TIMINGS=1 set in the worker env file; NOT deployed or verified live until the worker restarts on an empty queue. Step 4 (timings read after 24 h) and gate trims stay open."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer of docs/research/merge-queue-profile-2026-11-11.md. Done when the ordered plan steps 1 to 4 are in and the median gap between queued items is under 90 s.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Stale red hold cleared and red_hold made self-healing (ops repo, tools/mergeq.py)
- [x] #2 Only the ci workflow of main holds the queue; desktop-electron reported, not blocking
- [x] #3 Auto-Done follow-up push counted as an in-flight head or folded into the next merge
- [ ] #4 Measure with MERGE_TIMINGS=1 for a day, then decide gate trims (a, b, e)
<!-- AC:END -->

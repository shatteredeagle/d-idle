---
id: TASK-313
title: 'qprofile: merge queue profile and safe speedups'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T17:14:00Z'
updated_date: '2026-11-11T17:56:00Z'
labels:
  - ops
  - research
dependencies: []
priority: medium
ordinal: 216000
card: "qprofile"
owns:
  - "docs/research/merge-queue-profile-2026-11-11.md"
exclusive: []
contracts: []
result: "Read-only profile written; per-phase seconds are estimates (MERGE_TIMINGS off); not verified live; go test timing job skipped (gate active)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Read-only analysis of the merge queue. Consumer: queuespeed (TASK-312).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Profile written with numbers and ordered plan
- [x] #2 Consumer card queuespeed named
<!-- AC:END -->

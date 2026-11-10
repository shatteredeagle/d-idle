---
id: TASK-241
title: 'killloop: where the seconds per kill go, ranked, with the existing fixes'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T01:36:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 180000
card: "killloop"
owns:
  - "docs/killloop.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fun killer 3 (3 kills a minute, 12 of 18 s waiting). Done when: ranked table and area xp/h in docs/killloop.md.

result: analysis only, no code change; the sandbox binary lacked TASK-210 and packfight (dd208fa9) already on main; log numbers verified from the sandbox log, speedup estimate UNVERIFIED live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Rank the waits per kill from the sandbox log
- [x] #2 Report xp/h per area and compare with d2d pacing docs
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

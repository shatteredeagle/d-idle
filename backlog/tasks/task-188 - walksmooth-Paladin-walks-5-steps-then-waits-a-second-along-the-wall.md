---
id: TASK-188
title: 'walksmooth: Paladin walks 5 steps then waits a second along the wall'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T15:20:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: high
ordinal: 154000
card: "walksmooth"
owns:
  - "internal/nav/"
  - "internal/d2dinput/"
  - "docs/walksmooth.md"
  - "docs/probes/walksmooth/"
exclusive: []
contracts: []
result: "Offline fix (stop-short waypoint advance, click lookahead, run-belief heal), sim before/after, NOT verified live; see docs/walksmooth.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User feedback 2026-11-08: stop-and-go walk out of the start area. Done when smooth.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 root cause found with evidence (log evidence V, mechanism U: realtime debug run was refused)
- [x] #2 fix with regression test (two classes)
- [x] #3 before/after movement smoothness measured (offline sim only, not live)
<!-- AC:END -->

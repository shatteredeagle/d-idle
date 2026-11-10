---
id: TASK-110
title: 'run4h: 4-hour unattended realtime run of the play-session command'
status: Done
assignee:
  - live-agent
created_date: '2026-11-07T21:56:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - ops
  - drill
  - ladder
milestone: m-4
dependencies:
  - TASK-106
priority: medium
ordinal: 103000
card: "run4h"
owns:
  - "docs/probes/run4h.md"
exclusive:
  - "instance:i4"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: direction review section 7 step 4. Done when: docs/probes/run4h.md has the verdict (needs the live lab, the coordinator schedules it).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One 4-hour realtime run without a restart loop; record from the session record card; writes after start 0
- [ ] #2 If it cannot finish, the smoke card is what fixes it; result says so
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed as superseded in the 2026-11-10 true-up: folded into the awaydrill card (see TASK-248) and the sessions card.
<!-- SECTION:NOTES:END -->

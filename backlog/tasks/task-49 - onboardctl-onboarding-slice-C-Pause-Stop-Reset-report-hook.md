---
id: TASK-49
title: 'onboardctl: onboarding slice C: Pause, Stop, Reset, report hook'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-06T16:20:00Z'
updated_date: '2026-11-10T17:11:00Z'
labels:
  - watch
  - feature
  - onboarding
  - parked
milestone: m-1
dependencies: []
priority: medium
ordinal: 49000
card: "onboardctl"
owns:
  - "internal/watch/control.go"
  - "internal/watch/control_test.go"
  - "docs/onboarding-controls.md"
exclusive: []
contracts: []
result: "Superseded by TASK-27 (2026-11-10 true-up). Pause, Stop and Reset not carried. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Superseded (2026-11-10 true-up): the report-preview hook moves into TASK-27 (exportable feedback record). Pause, Stop and Reset are not in the beta exit criteria of docs/research/project-review-2026-11-10.md section 5.1 and are not carried by the three kept cards; reopen as a new card if the friend beta needs Reset.
Size: S
Why: m-1 beta re-cut, see docs/research/project-review-2026-11-10.md sections 3.3 and 5.
Done when: superseded by TASK-27.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Pause/resume, Stop and Reset routes on the answer listener, origin-checked, typed word for Reset
- [ ] #2 Report preview page hook to TASK-27 (bundle) and text for TASK-26 (guide)
- [ ] #3 Tests for each route and the gate
<!-- AC:END -->

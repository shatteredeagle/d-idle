---
id: TASK-49
title: 'onboardctl: onboarding slice C: Pause, Stop, Reset, report hook'
status: Proposed
assignee: []
created_date: '2026-11-06T16:20:00Z'
labels:
  - onboarding
  - watch
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice C of docs/research/onboarding-ux-design.md (approved 2026-11-06): Pause and Stop in the controls sheet, Reset behind a typed word (deletes hero, rules, journal, not game files or passcode), report preview and bundle hook to TASK-27, guide hook to TASK-26. About 7 h. Uses the passcode gate of onboardA (internal/access).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Pause/resume, Stop and Reset routes on the answer listener, origin-checked, typed word for Reset
- [ ] #2 Report preview page hook to TASK-27 (bundle) and text for TASK-26 (guide)
- [ ] #3 Tests for each route and the gate
<!-- AC:END -->

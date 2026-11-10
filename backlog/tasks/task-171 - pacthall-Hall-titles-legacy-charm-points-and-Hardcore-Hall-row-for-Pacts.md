---
id: TASK-171
title: 'pacthall: Hall titles, legacy charm points and Hardcore Hall row for Pacts'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-08T03:34:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - feature
  - parked
milestone: m-4
dependencies:
  - TASK-166
  - TASK-135
  - TASK-136
priority: low
ordinal: 147000
card: "pacthall"
owns:
  - "internal/watch/pacthall.go"
  - "internal/watch/pacthall_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Hall row lists Pacts, ranks and highest heat
- [ ] #2 Charm points +1 per three Pacts at rank 3 or higher at retire
- [ ] #3 Titles at rank 5; separate Hardcore row
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: first Pacts need Andariel killed on a live hero.
<!-- SECTION:NOTES:END -->

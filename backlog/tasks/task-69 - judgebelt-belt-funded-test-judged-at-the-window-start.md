---
id: TASK-69
title: 'judgebelt: belt funded test judged at the window start'
status: Done
assignee: []
created_date: '2026-11-07T13:12:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - gate
dependencies: []
ordinal: 67000
card: "judgebelt"
owns:
  - "internal/d2dsoakstat/invariants.go"
  - "internal/d2dsoakstat/invariants_test.go"
  - "docs/contracts/soak-gates.md"
  - "docs/decisions.md"
exclusive: []
result: "Offline only: unit tests in internal/d2dsoakstat (seed 17 case and contrast); not verified live. The funded logic lives in the public d2dsoakstat, not in the private judge."
contracts:
  - "soak-gates"
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Decisions entry 2026-11-07 added
- [x] #2 d2dsoakstat funded test at the window's first minute, tests for seed 17 case and the contrast
- [x] #3 soak-gates BELT funded wording and scenarios updated
<!-- AC:END -->

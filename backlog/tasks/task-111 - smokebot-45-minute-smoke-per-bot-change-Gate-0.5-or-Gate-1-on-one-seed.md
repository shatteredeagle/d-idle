---
id: TASK-111
title: 'smokebot: 45-minute smoke per bot change (Gate 0.5 or Gate 1 on one seed)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - ops
  - feature
  - gate
milestone: m-4
dependencies: []
priority: high
ordinal: 104000
card: "smokebot"
owns:
  - "tools/smoke.sh"
  - "tools/smoke-worker.sh"
  - "cmd/d2smoke"
  - "internal/smoke"
  - "data/smoke"
  - "docs/soaks/smoke-*"
  - "docs/smoke.md"
exclusive: []
contracts: []
result: "Built and unit-tested offline: tools/smoke.sh, tools/smoke-worker.sh, cmd/d2smoke + internal/smoke judge, data/smoke/rules.json, docs/smoke.md. NOT run live (U); AC1 means the command exists and prints the verdict line, first live run is the checklist in docs/smoke.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: direction review section 7 step 5 (decision D1). Done when: a bot merge can get a one-line verdict instead of a 24-hour wave.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 One command runs Gate 0.5 (or Gate 1, one seed, 45 min) on a branch and prints a one-line verdict
- [x] #2 docs/smoke.md states it is the regression floor while Gate 2 waves are paused (ADR 0017); built vs verified live
- [ ] #3 leakcheck 0; tools/ci/check.sh
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

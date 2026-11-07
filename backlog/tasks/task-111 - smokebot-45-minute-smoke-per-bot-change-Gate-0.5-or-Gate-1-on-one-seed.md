---
id: TASK-111
title: 'smokebot: 45-minute smoke per bot change (Gate 0.5 or Gate 1 on one seed)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
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
  - "tools/smoke"
  - "docs/smoke.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: direction review section 7 step 5 (decision D1). Done when: a bot merge can get a one-line verdict instead of a 24-hour wave.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One command runs Gate 0.5 (or Gate 1, one seed, 45 min) on a branch and prints a one-line verdict
- [ ] #2 docs/smoke.md states it is the regression floor while Gate 2 waves are paused (ADR 0017); built vs verified live
- [ ] #3 leakcheck 0; tools/ci/check.sh
<!-- AC:END -->

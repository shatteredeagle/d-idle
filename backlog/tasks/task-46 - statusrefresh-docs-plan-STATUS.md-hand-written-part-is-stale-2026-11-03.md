---
id: TASK-46
title: 'statusrefresh: docs/plan/STATUS.md hand-written part is stale (2026-11-03)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-06T16:02:00Z'
updated_date: '2026-11-27T20:31:00Z'
labels:
  - docs
  - chore
dependencies: []
priority: medium
ordinal: 46000
card: "statusrefresh"
owns:
  - "docs/plan/STATUS.md"
exclusive: []
contracts: []
lane: next
result: "Hand-written part of docs/plan/STATUS.md rewritten from ADR 0016/0017, the Current truth block and plan.py; every claim links its owner doc, no copied numbers; generated block unchanged; tools/ci/check.sh --hygiene-only passes. Docs only, no live check."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The hand-written top still ranks by goals G-A..G-F and says no Gate runs; the ladder (ADR 0016), Gate 2 g2d/g2e/g2f, the tester build and the watch screen are missing. The generated block is untouched.

Done when: Hand-written part rewritten from ADR 0016, docs/decisions.md Current truth and the backlog; every number...; Generated block unchanged; tools/ci/check.sh --hygiene-only passes (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Hand-written part rewritten from ADR 0016, docs/decisions.md Current truth and the backlog; every number links to its owner doc or is omitted
- [x] #2 Generated block unchanged; tools/ci/check.sh --hygiene-only passes
<!-- AC:END -->

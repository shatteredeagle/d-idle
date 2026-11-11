---
id: TASK-345
title: 'prverify2: second-pass verification of upstream PR candidates Lightning last unit (r3) and Cold Enchanted length (r4)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T22:16:00Z'
labels:
  - d2d
  - readonly
dependencies: []
priority: high
ordinal: 228000
card: "prverify2"
owns:
  - "docs/upstream/verification-2026-11-11.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Second independent Opus pass after the prep applied the first pass's fixes (TASK-340, branch agent/prverify; prep TASK-332 agent/prprep2). Consumer: TASK-45 (upstream-next, the PR-posting card). Done when the doc carries a second-pass verdict per candidate with post-ready title, body, branch and base.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each first-pass finding re-checked as resolved or not on the r3 and r4 heads
- [ ] #2 Every commit of both PR branches builds alone, as pushed and rebased on current upstream main
- [ ] #3 Both differentials re-run on clean worktrees, all seeds, parent and claim; ctest on the claims against upstream main
- [ ] #4 Diff hygiene, upstream contribution rules, upstream overlap since 62732598, single topic checked
- [ ] #5 Second-pass section with verdicts and post-ready data in docs/upstream/verification-2026-11-11.md
<!-- AC:END -->

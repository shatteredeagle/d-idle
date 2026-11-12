---
id: TASK-360
title: 'cutfix: checklist accepts an empty UPDATE_FEED (CI red)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T14:33:00Z'
labels:
  - container
  - bug
dependencies: []
priority: high
ordinal: 235000
card: "cutfix"
result: "CI red cause: the new UPDATE_FEED check treated an empty feed file (a lone newline in the test packs) as a bad URL; test/logic bug, not a leak. Fixed; friendpack tests pass in a clean worktree and the real pack still passes the checklist. Not verified in CI itself."
owns:
  - "tools/friendpack/checklist.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 friendpack tests green in a clean checkout
<!-- AC:END -->

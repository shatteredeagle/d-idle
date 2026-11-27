---
id: TASK-982
title: >-
  bsmchk: boardsm checker: questspec rules 12 to 14 in the merge gate (card 8 of
  t
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:57:00Z'
updated_date: '2026-11-27T19:42:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: low
ordinal: 470000
card: "bsmchk"
owns:
  - "docs/design/boardsm-checker.md"
  - "internal/questspec"
  - "internal/task/guardcodes_test.go"
  - "docs/questspec.md"
exclusive: []
contracts: []
result: "Rules 12 (CheckOrder), 13 (CheckLatched, chain field latched), 14 (CheckGuardCodes, task/gate.go scan) with positive and negative tests; 12 and 13 in TestShippedChains, shipped data clean, no allow entries. Offline, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Recommendation of the quest-board review (docs/design/fable-review-2026-11-21/quest-board.md section 5 card 8, appendix B.3): questspec rules 12 to 14 checked in TestShippedChains. No card exists (grep of internal/questspec finds no such rules at this commit; U if added under another name). Cause class: B and H (stored-run invariants). Risk: low (allow-list). Enable criteria: rules in CI.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Rules 12 to 14 in TestShippedChains with an allow-list
<!-- AC:END -->

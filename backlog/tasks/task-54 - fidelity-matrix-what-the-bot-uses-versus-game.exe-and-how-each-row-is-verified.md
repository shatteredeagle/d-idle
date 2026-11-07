---
id: TASK-54
title: 'fidelity-matrix: bot use versus game.exe, how each row is verified'
status: Done
assignee: []
created_date: '2026-11-06T22:38:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - docs
  - research
  - fidelity
  - gate
milestone: m-2
dependencies: []
priority: medium
ordinal: 52000
card: "fidelity-matrix"
owns:
  - "docs/research/fidelity-matrix.md"
  - "docs/research/fidelity-probe-design.md"
exclusive: []
contracts: []
result: "docs only, not verified live: matrix of 78 rows (18 OD, 21 DEV, 9 RO, 4 DT, 7 DM, 13 NONE, 6 MODEL), same-save probe design, tasks TASK-55 and TASK-56 (Proposed)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User question 2026-11-06: do we have a plan to verify d2d is 1:1 with game.exe for everything the bot uses. Docs and tasks only.

Done when: docs/research/fidelity-matrix.md: one row per bot-used behaviour with current verification, method, cost,...; docs/research/fidelity-probe-design.md: same-save probe design (read-only, no cheats) with harness shape,...; ranked first slice, slice plan (at most 6, with hours) and tasks for the first two slices (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/research/fidelity-matrix.md: one row per bot-used behaviour with current verification, method, cost, risk, priority, totals
- [x] #2 docs/research/fidelity-probe-design.md: same-save probe design (read-only, no cheats) with harness shape, scenarios, pass criteria
- [x] #3 ranked first slice, slice plan (at most 6, with hours) and tasks for the first two slices
<!-- AC:END -->

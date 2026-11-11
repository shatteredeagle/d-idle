---
id: TASK-329
title: 'runstate: infer run or walk from exact stamina deltas'
status: Proposed
assignee:
  - pro-runner
created_date: '2026-11-11T19:21:00Z'
labels:
  - bot
  - bot-fix
  - fidelity
dependencies: []
priority: medium
ordinal: 223000
card: "runstate"
owns:
  - "internal/d2dinput/run.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: fid-stamina.md: drain and regen per frame are exact (V run, 0 mismatch in 80000 cases), so the toggle is readable from two stamina samples. Done when: the belief is corrected from the numbers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 classify each stamina step as run drain, walk regen or stand regen from the V run rules
- [ ] #2 runBelieved flips only on a mismatch
- [ ] #3 unit test
<!-- AC:END -->

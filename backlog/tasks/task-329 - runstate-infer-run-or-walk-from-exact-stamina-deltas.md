---
id: TASK-329
title: 'runstate: infer run or walk from exact stamina deltas'
status: Done
assignee:
  - pro-runner
created_date: '2026-11-11T19:21:00Z'
updated_date: '2026-11-12T01:38:00Z'
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
  - "internal/d2dinput/controller.go"
exclusive: []
contracts: []
result: "stamina steps classified from the fid-stamina rules (fall = run, walking-rate rise = walk, no change = unknown); belief flips only on a mismatch; logged as runObs/runFix; unit-tested offline, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: fid-stamina.md: drain and regen per frame are exact (V run, 0 mismatch in 80000 cases), so the toggle is readable from two stamina samples. Done when: the belief is corrected from the numbers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 classify each stamina step as run drain, walk regen or stand regen from the V run rules
- [x] #2 runBelieved flips only on a mismatch
- [x] #3 unit test
<!-- AC:END -->

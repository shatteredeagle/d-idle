---
id: TASK-735
title: >-
  pgdigest: assertion digest D1; determinism compares D1 hard and the state
  digest as drift
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:45:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 342000
card: "pgdigest"
owns:
  - "internal/scenario/determinism.go"
  - "cmd/d2scenario/cmds.go"
  - "internal/scenario/assertdigest.go"
  - "internal/scenario/assertdigest_test.go"
  - "internal/scenario/result.go"
exclusive: []
contracts: []
result: "Assertion digest D1 (internal/scenario/assertdigest.go: run key plus per-assertion kind and bucket, discrete kinds by value) is written by exec-one; determinism gates on verdict + D1 (hard) and reports the whole-state digest as drift only (Q1). Old results are compared by recomputing D1 from their stored asserts. Verdict changes: none for any scenario; determinism of a pair now also fails on a changed D1. Offline evidence on the 2026-11-19 run records: cain.stones, cain.cage, cain.scroll(Sor) pairs deterministic with state-digest drift only; chicken.recovery pair fails hard on D1 (same RED verdict, different facts). Tests -race -count=20; not run live."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design section 2.6, 7 card 2 (user Q1: verdict + assertion digest gate; full-state digest is a drift report). Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 D1 = scenario, class, seed and per-assertion facts (discrete by value, ranges by bucket)
- [x] #2 determinism gates on verdict + D1; whole-state digest reported as drift only
- [x] #3 must-fail test: gold 700 vs 712 stays equal, quest word differs fails
- [x] #4 existing scenario verdicts unchanged (listed)
<!-- AC:END -->

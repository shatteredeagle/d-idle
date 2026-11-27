---
id: TASK-754
title: >-
  pgnightly: one nightly scheduler job: T2, T3, pairs, one T4; smoke stage after
  each push
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:51:00Z'
updated_date: '2026-11-27T21:50:00Z'
labels:
  - ops
  - feature
dependencies: []
priority: medium
ordinal: 355000
card: "pgnightly"
owns:
  - "docs/process/pg-ops-pgnightly.md"
  - "tools/provingground/nightly.sh"
  - "tools/provingground/nightly.json"
  - "internal/scenario/nightly.go"
  - "internal/scenario/nightly_test.go"
  - "cmd/d2scenario/nightly.go"
  - "cmd/d2scenario/nightly_test.go"
  - "cmd/d2scenario/main.go"
  - "docs/probes/pg-nightly-earned-handoff-2026-11-27.md"
exclusive: []
contracts: []
result: "Repo side done, offline only: the nightly run set is data (tools/provingground/nightly.json, window 23:00-07:00 EST, at most 3 jobs, nice 19, hidden lab slots, caps per stage, smoke report-only for 7 clean days), validated and expanded by internal/scenario/nightly.go, printed by d2scenario nightly check|plan, run by nightly.sh (dry run default, refuses outside the window). Tests with fakes and the shipped data green; no night has run, the scheduler timer and the after-push smoke hook are ops-side (docs/probes/pg-nightly-earned-handoff-2026-11-27.md). Not verified live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design 4.6, 7 card 15. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Nightly job runs T2, T3, pairs and one T4 in the standing worker-host window 23:00-07:00 EST only, at most 3 runs, niced (Q4)
- [x] #2 smoke set report-only for 7 clean days, then the user decides on blocking (Q2)
<!-- AC:END -->

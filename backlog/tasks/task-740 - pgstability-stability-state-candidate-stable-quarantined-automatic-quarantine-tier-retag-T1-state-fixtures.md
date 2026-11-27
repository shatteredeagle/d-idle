---
id: TASK-740
title: >-
  pgstability: stability state candidate/stable/quarantined, automatic
  quarantine, tier retag, T1 state fixtures
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:47:00Z'
updated_date: '2026-11-27T06:13:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 346000
card: "pgstability"
owns:
  - "internal/scenario/stability.go"
  - "internal/scenario/stability_test.go"
  - "internal/scenario/scenario.go"
  - "tools/provingground/scenarios/tiers.json"
exclusive: []
contracts: []
lane: next
result: "Stability state derived from the results ledger: 5 equal runs (verdict and assertion digest, one class and seed, unchanged bot commit and d2d hash) make a scenario stable; a same-binary flip quarantines it by tool; only a person releases it, with a root-cause note, and it then needs 5 clean runs again. tiers.json gives every scenario a proposed tier (32 from the design catalogue, the rest mapped: fast and full to T3, nightly to T4) and LoadDir skips it. Tested offline incl. a must-fail flip test (mutation-checked); not verified live, no stable scenario yet, so no T1 state fixtures were cut (LastGreen names the run to cut from). Not build-specific."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design 2.7 and 3, 7 card 6 (Q7: auto-quarantine, a person releases). Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 5 equal runs make stable; a flip on unchanged commit and d2d hash quarantines by tool
- [x] #2 tier per scenario from the catalogue
- [x] #3 must-fail test for the flip rule
<!-- AC:END -->

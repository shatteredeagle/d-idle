---
id: TASK-655
title: 'survhold3: starved caster yield, risk route closed at the k limit, holdout 3'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-18T22:46:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 325000
card: "survhold3"
owns:
  - "internal/survival"
  - "docs/probes/survival-holdout3-2026-11-18.md"
exclusive: []
contracts: []
result: "Risk route closed at the k limit (riskMaxK) and caster yieldMinMinutes/starveValueMul as data, must-fail tests on two builds; Sorceress seeds 101/103 re-run on the builder (103 recovered: margin 22.9k against legacy 16.9k; 101 12.0k against 16.1k; writes after start 0): INCONCLUSIVE with 2 pairs; seed 107 and the Paladin waves not run (builder yielded to merge gates); no default changed"
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Risk route closed at k limit and caster yield/starve data fields, must-fail tests on two builds
- [ ] #2 Sorceress 3-seed screen and 2 more Paladin seeds on the builder long lane, writes after start 0
- [x] #3 docs/probes/survival-holdout3-2026-11-18.md with per-class verdict
<!-- AC:END -->

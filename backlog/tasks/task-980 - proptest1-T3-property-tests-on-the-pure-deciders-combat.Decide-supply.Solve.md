---
id: TASK-980
title: >-
  proptest1: T3 property tests on the pure deciders (combat.Decide,
  supply.Solve,
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:56:00Z'
updated_date: '2026-11-27T05:58:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: low
ordinal: 468000
card: "proptest1"
owns:
  - "docs/design/proptest-t3.md"
  - "internal/combat/decide_prop_test.go"
  - "internal/supply/solve_prop_test.go"
  - "internal/arbiter/decide_prop_test.go"
exclusive: []
contracts: []
result: "Property tests on combat Decide, supply Solve and the arbiter Decide, two builds each, fixed seeds; every property caught a one-line mutation; no bug found; Solve is gold-monotone only for min-before-target policies (noted). Offline, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Cause classes: B, C, G at unit-test cost. Roadmap row T3 (docs/research/fable-logic-review-2026-11-25.md 5.3). Random-input properties: a reflex always wins; more potions never worsen the verdict; no intent starves; a refusal's need, once met, admits. One package per PR (combat, supply, arbiter first; admission after admit1). Two builds as input generators. Risk: none. Enable criteria: tests in CI.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Properties for combat.Decide, supply.Solve and arbiter.Decide with two builds
<!-- AC:END -->

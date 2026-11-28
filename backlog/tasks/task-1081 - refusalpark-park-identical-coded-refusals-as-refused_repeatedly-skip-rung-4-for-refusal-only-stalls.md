---
id: TASK-1081
title: 'refusalpark: park identical coded refusals as refused_repeatedly, skip rung 4 for refusal-only stalls'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T00:05:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 518000
card: "refusalpark"
knob_net: owner-ok
lane: next
owns:
  - "internal/questboard/"
  - "data/questboard/"
  - "internal/bot/questboard.go"
  - "internal/bot/questauto.go"
  - "internal/bot/invariants.go"
  - "internal/bot/loop.go"
  - "internal/bot/failsafe_test.go"
  - "internal/env/tuning_quests.go"
  - "data/darklaunch.json"
  - "docs/knobs.md"
  - "docs/questboard.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. quests.refusalPark (default 0): refusal.json parkAfter 3 identical coded refusals park as refused_repeatedly (code and text in reason and Return screen); rung 4 skipped for refusal-only stalls. Tests: internal/questboard refusal_test.go (two classes), internal/bot failsafe_test.go."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User approved the knob 2026-11-27 (popup). Done when the tests pass with two classes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Knob quests.refusalPark (default 0) parks N identical coded refusals as refused_repeatedly with code and text
- [x] #2 Mixed codes, uncoded refusals and knob 0 behave as today
- [x] #3 Failsafe rung 4 skipped only for a refusal-only stall
- [x] #4 Ledger row, knobs.md regenerated, checks green
<!-- AC:END -->

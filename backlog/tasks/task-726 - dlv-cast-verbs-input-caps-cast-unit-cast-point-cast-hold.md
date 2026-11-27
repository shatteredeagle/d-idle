---
id: TASK-726
title: 'dlv-cast-verbs: input caps cast-unit, cast-point, cast-hold'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:43:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 349000
card: "dlv-cast-verbs"
branch: "agent/castverbs"
owns:
  - "internal/input/"
  - "internal/d2dcombat/"
  - "internal/d2dinput/"
  - "data/input/caps/"
  - "data/input/caps_test.go"
  - "docs/design/input-cast-verbs.md"
  - "tools/provingground/scenarios/input-scenarios.json"
  - "tools/provingground/scenarios/tiers.json"
  - "tools/provingground/scenarios/behaviours.json"
exclusive:
  - "fork-branch:didle/cast-verbs"
contracts: []
lane: next
result: "Built offline, not verified live. Input caps cast-unit, cast-point, cast-hold with a cast request and RunCast that sends the verb only with input.planner on and the cap listed, falls back once only when the verb was not sent, never casts twice; d2d controller cast verbs; combat Deliver reads hand, target and press from the capability row and carries the corpse unit; candidate caps file and pending scenario input.cast-point-lands; tests on Sorceress, Paladin and Barbarian rows; class and skill literals added 0 (inventory scope untouched); no combat driver ships here, so no combat.delivery knob (the input knob input.planner is the rollback). Not done: the Fighter cast section still calls CastAt and Click (file not owned), the fork differential and the hold verb (described as a fork patch in docs/design/input-cast-verbs.md)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Delivery builds an input.Action per cast (verb, key, click, verify=expectation); corpse casts carry the corpse unit; Fighter stops calling Click and CastAt directly. Fused cast verb allowed as input only if a differential equals select-then-cast (Q3). No new pixel work (Q10). Depends on dlv-expect.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenario input.cast-point-lands: a point spell lands at the aimed cell
- [ ] #2 Differential fused verb vs select-then-cast
- [x] #3 Knob input.planner (exists, default 0)
- [x] #4 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #5 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #6 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

---
id: TASK-726
title: 'dlv-cast-verbs: input caps cast-unit, cast-point, cast-hold'
status: Done
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
result: "Built offline. Fork differential run 2026-11-27 in two headless d2d, NOT verified live: the fused cast verb equals select then a press with the old hand put back, 0 differences on unit, point, left hand, corpse and four refusals, and the point spell lands within one cell; a hold verb was added to the private fork and equals a held left button. The fork branch lists cast-unit, cast-point and cast-hold. Since then (commit a13fc6141, on main): the controller has CastHold (internal/d2dinput/casthold.go) and the Fighter casts through Deliver with input.planner on, the old CastAt and Click only with the knob off or no capability row (castdeliver.go, castdeliver_test.go on a Sorceress and a Paladin). Still open: the scenario input.cast-point-lands has not run (needs the fork build, live). Class and skill literals added in Go: 0 (rows from the capability table). closecards3: status set to Done, merged on main; the live scenario box above stays open (see docs/live-checks.md at merge)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Delivery builds an input.Action per cast (verb, key, click, verify=expectation); corpse casts carry the corpse unit; Fighter stops calling Click and CastAt directly. Fused cast verb allowed as input only if a differential equals select-then-cast (Q3). No new pixel work (Q10). Depends on dlv-expect.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenario input.cast-point-lands: a point spell lands at the aimed cell
- [x] #2 Differential fused verb vs select-then-cast
- [x] #3 Knob input.planner (exists, default 0)
- [x] #4 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #5 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #6 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

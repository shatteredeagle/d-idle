---
id: TASK-1038
title: >-
  casthold: d2d controller hold verb and the fighter casts through the input
  planner
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T07:51:00Z'
updated_date: '2026-11-27T08:19:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 502000
card: "casthold"
lane: now
owns:
  - "internal/d2dinput/"
  - "internal/d2dcombat/"
  - "internal/d2dctl/fake/"
  - "internal/combat/fighter.go"
  - "internal/combat/castdeliver.go"
  - "internal/combat/castdeliver_test.go"
  - "docs/design/input-cast-verbs.md"
exclusive: []
contracts: []
result: "Built offline, not verified live. The d2d controller holds a skill through the fork hold verb and polls its status until the hold ends, bounded by the hold plus half a second; only a refused start falls back to one cast, a started hold never casts twice. With input.planner on the Fighter sends every cast that has a capability row through Deliver (hand, target and press from the row), with the knob off its old calls are unchanged; the combat rig built for drills now shares the planner with the controller. Tests on Sorceress Fire Bolt and Paladin Zeal and Smite, plus Paladin and Barbarian hold rows; nothing build-specific in Go, the data is the capability table and the profile hold time. Finding: the d2d skill command sends the word left, which the fork reads as the right hand (source read). Live check open."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Follow-up of TASK-726 (castverbs) and forkcaps. Done when the controller has CastHold against the fork skillhold verb and the Fighter casts through Deliver behind input.planner.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 d2dinput Controller implements input.HoldCaster with cmd skillhold and polls cmd skillhold status until the hold ends, bounded by the hold plus a margin; offline fake tests
- [x] #2 A started hold is never followed by a second cast (status errors and timeouts do not fall back); a refused start falls back to one cast
- [x] #3 The Fighter cast section goes through the backend Deliver path with input.planner on, using the capability row of the skill; with input.planner 0 its calls are unchanged
- [x] #4 Tests on two builds: a caster (Sorceress) and a melee hold skill (Paladin Zeal)
- [ ] #5 Live check: hidden instance with the forkcaps build, input.planner=1, a Paladin holds Zeal and the log shows input path action=cast-hold path=verb
<!-- AC:END -->

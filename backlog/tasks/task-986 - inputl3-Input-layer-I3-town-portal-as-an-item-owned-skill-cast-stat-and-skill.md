---
id: TASK-986
title: 'inputl3: Input layer I3: town portal as an item-owned skill cast; stat and skill'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-26T03:58:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 474000
card: "inputl3"
branch: "agent/forkcaps"
owns:
  - "docs/design/input-i3.md"
  - "internal/input/useitem.go"
  - "internal/input/useitem_test.go"
  - "internal/d2dctl/use_verb.go"
  - "internal/d2dctl/use_verb_test.go"
  - "internal/d2dctl/fake/fake.go"
  - "internal/d2dinput/useitem.go"
  - "internal/combat/portalverb.go"
  - "internal/combat/portalverb_test.go"
  - "internal/combat/fighter.go"
  - "internal/town/portalverb.go"
  - "internal/town/portalverb_test.go"
  - "internal/town/portalverb_unit_test.go"
  - "internal/town/portalverb_export_test.go"
  - "internal/town/trip.go"
  - "data/input/caps/"
  - "data/input/caps_test.go"
  - "docs/design/input-cast-verbs.md"
  - "backlog/tasks/task-726 - dlv-cast-verbs-input-caps-cast-unit-cast-point-cast-hold.md"
exclusive: []
contracts: []
result: "Bot side built offline, NOT verified live. Town portal casts send the use-item verb when input.planner is on and the build lists input.use-item; verified by the portal object, the scroll gone or the tome charges falling; a sent verb with no effect is never followed by the click. Fork differential run 2026-11-27 in two headless d2d: town portal by the verb equals the pack right-click, scroll, tome twice and three refusals, 0 differences; the private fork branch now lists input.use-item and the caps set is in the repo. Stat and skill points were already on verbs; their differential has not run, so criterion one stays open. Nothing is build-specific; tests on four classes."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Recommendation R17 of the 2026-11-19 review (07-input-layer table I3; ADR 0021 verb, then key, then verified click). inputlayer (I1, I2) is done; I3 to I7 have no card. Each verb promoted needs its own differential. Cause class: F and the click-path bounce incidents. Risk: low to medium. Enable criteria: input.planner stays default 0 until the drill passes (ledger row input.planner).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Town portal and points on verbs with a differential each
- [x] #2 Bot side: town portal by the use-item verb behind input.planner and input.use-item, click fallback only on a send error, tests on two or more classes
- [x] #3 Differential checklist for the town portal and the points verbs written for the fork (docs/design/input-i3.md section 3)
<!-- AC:END -->

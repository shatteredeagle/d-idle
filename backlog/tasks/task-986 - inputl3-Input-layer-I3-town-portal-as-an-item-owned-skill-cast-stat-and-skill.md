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
exclusive: []
contracts: []
result: "Offline only, NOT verified live. Town portal casts (combat safety portal and the town trip) send the use-item verb, the message the pack right-click sends, when input.planner is on and the build lists input.use-item; verified by the portal object, the scroll gone or the tome charges falling; a sent verb with no effect is never followed by the click. Stat and skill points were already on verbs. Nothing is build-specific; tests on four classes. The two differentials are written as a fork checklist in the design doc, not run; no build lists input.use-item yet, so the change is dark."
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

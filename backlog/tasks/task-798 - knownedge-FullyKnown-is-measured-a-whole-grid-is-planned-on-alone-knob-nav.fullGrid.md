---
id: TASK-798
title: >-
  knownedge: FullyKnown is measured; a whole grid is planned on alone (knob
  nav.fullGrid)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T02:31:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 367000
card: "knownedge"
owns:
  - "internal/nav/fullgrid_test.go"
  - "internal/nav/roam.go"
  - "internal/nav/nav.go"
  - "internal/env/tuning_nav.go"
  - "internal/d2dworld/world.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Offline only (not verified live): FullyKnown is measured from the grid (known cells equal the level rectangle) behind knob nav.fullGrid (default 0, behaviour unchanged); with it a whole grid refuses a goal outside it (no walk to a known edge) and a partial grid keeps the optimistic walk. Must-fail tests on the Paladin and Sorceress Tower Cellar 5 walks (internal/nav/fullgrid_test.go), -race -count=20 clean. The gridwhole probe (merged) showed d2d grids are whole at arrival, so the knob turns a measured fact into the rule; the mouthgoal_test comment on a known grid edge is corrected. Scenarios act1.forgotten.countess not run with the knob (U). Build-specific: none, the rule is generic."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: review docs/design/fable-review-2026-11-22/map-knowledge.md section 5 card 2, after the gridwhole probe. Done when the tests pass and the knob is documented.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 FullyKnown is measured from the grid (known cells equal the level rectangle) behind nav.fullGrid (default 0)
- [x] #2 A whole grid refuses a goal outside it instead of walking to a known edge; a partial grid keeps the optimistic walk
- [x] #3 Must-fail tests for the Paladin and Sorceress walks (internal/nav/fullgrid_test.go)
- [x] #4 d2dworld logs the known cell count of every arrival
<!-- AC:END -->

---
id: TASK-799
title: >-
  fogtrail: Map panel shows walked ground and the objective marker, forgets it
  per game
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T02:35:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 367000
card: "fogtrail"
owns:
  - "internal/senses/objective.go"
  - "internal/senses/objective_test.go"
  - "internal/watch/web/panels/map"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/nav/roam.go"
  - "internal/nav/objectiveview_test.go"
  - "cmd/d2bot/sensesfeed.go"
  - "docs/mapfog.md"
exclusive: []
contracts: []
result: "Offline only (stub-DOM and Go tests, no live look, no screenshot: U): /api/senses gains an additive objective key (the goal of the walk in progress, not fogged, current area only); the Map draws a ring with a dot over the fog and its note says the ring is the goal (phone-friendly canvas, no inner scroll box); the walked-ground fog is forgotten per game (Publisher.NewGame through ForgetCoverage) because a new game lays the same area id out again. Decisions documented in docs/mapfog.md: walked ground is the visited-block set, one key per 4x4 subtiles (at most about 8400 per level), not persisted over a bot restart. Must-fail tests: internal/watch/testdata/map_panel.js, internal/senses/objective_test.go, internal/watch/senses_test.go, internal/nav/objectiveview_test.go. Not done: a label per objective kind. Build-specific: none."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: review docs/design/fable-review-2026-11-22/map-knowledge.md card 5 and Q5. Done when the viewer shows walked ground plus the objective marker.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 objective key in /api/senses (not fogged), current area only
- [x] #2 Map panel draws the objective ring over the fog, phone-friendly, no inner scroll box
- [x] #3 walked ground forgotten per game (Publisher.NewGame via ForgetCoverage); size and persistence decided in docs/mapfog.md
- [x] #4 must-fail UI test (map_panel.js) and Go tests
<!-- AC:END -->

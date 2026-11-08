---
id: TASK-195
title: 'mapreveal: later zones stay hidden on the minimap until found or unlocked'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T16:53:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies: []
priority: low
ordinal: 157000
card: "mapreveal"
owns:
  - "internal/senses"
  - "internal/watch/web"
  - "docs/senses-map.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User request 2026-11-08: the minimap should hide or fade out later zones and reveal a zone only once the hero has found it (visited, or its waypoint or exit seen) or the unlock ladder opens it; no spoilers of areas the hero has not met yet. The stitched neighbour map (TASK-189) draws the previous neighbours only; this card covers the world/atlas view and any area list: show unvisited zones as hidden or faded silhouettes, reveal on first visit, waypoint seen, or the matching unlock row; the waypoint atlas (TASK-160) uses the same rule. Done when the rule is data driven (per-area reveal trigger in data), tested with two classes, and checked live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 reveal rule defined as data: an area is shown once visited, its exit or waypoint was seen, or its unlock row opened; unknown areas are hidden or faded
- [ ] #2 minimap and atlas draw hidden areas as faded silhouettes or not at all, with no area name leaked before reveal (API too, not only the page)
- [ ] #3 tests with two classes: an unvisited area is absent from /api/senses and the atlas payload; reveal on first visit and on waypoint seen
- [ ] #4 live check on a real run: later zones stay hidden until entered
<!-- AC:END -->

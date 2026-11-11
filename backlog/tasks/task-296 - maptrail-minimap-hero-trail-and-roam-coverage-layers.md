---
id: TASK-296
title: 'maptrail: minimap hero trail and roam coverage layers'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T02:35:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 207000
card: "maptrail"
owns:
  - "internal/watch/web/panels/map"
  - "internal/senses"
  - "internal/nav/covview.go"
  - "internal/nav/covview_test.go"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/senses_stitch_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/senses-trail.json"
  - "internal/watch/testdata/senses-coverage.json"
  - "internal/nav/roam.go"
  - "cmd/d2bot/sensesfeed.go"
  - "cmd/d2bot/sensesfeed_test.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "tools/uishots/serve.go"
  - "docs/senses-map.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user cannot see where the hero runs or how the bot explores. Done when: the map panel draws a 30 s / 1 min / 5 min trail with stall dots and zone ticks, and an optional roam coverage grid. result: offline only (Go tests, node panel test); not verified live, no live bot seen. Trail keys `trail`, `trail_area`, `ts` and `coverage` are additive in /api/senses; schema string stays watch.senses/1 (additive keys, goldens senses-trail and senses-coverage added); panel manifest version 2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Hero trail ring in the senses publisher, additive trail keys in /api/senses, trail layer in the map panel
- [ ] #2 Roam coverage accessor, additive coverage key, Coverage toggle in the map panel
- [ ] #3 Tests: Go ring/payload goldens, node panel test; whole-repo gate green
<!-- AC:END -->

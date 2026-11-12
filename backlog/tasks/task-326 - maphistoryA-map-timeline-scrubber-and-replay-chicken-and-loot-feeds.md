---
id: TASK-326
title: 'maphistoryA: map timeline scrubber and replay, chicken and loot feeds'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:21:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 222000
card: "maphistoryA"
owns:
  - "internal/watch/web/panels/map"
  - "internal/senses"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/history.json"
  - "internal/watch/testdata/senses-trail.json"
  - "internal/bot/runner.go"
  - "internal/bot/goal.go"
  - "cmd/d2bot/senseshistory.go"
  - "cmd/d2bot/senseshistory_test.go"
  - "cmd/d2bot/funwire.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "docs/senses-map.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 GET /api/history samples ring (2 s, 60 min), panel timeline: ribbon, scrubber, replay 1x/4x/16x, readout
- [ ] #2 Feeds: chicken, portal, waypoint markers; loot walk-back activity
<!-- AC:END -->

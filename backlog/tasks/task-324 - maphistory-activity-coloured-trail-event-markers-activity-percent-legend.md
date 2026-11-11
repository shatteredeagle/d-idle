---
id: TASK-324
title: 'maphistory: activity-coloured trail, event markers, activity percent legend'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:11:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 221000
card: "maphistory"
owns:
  - "internal/watch/web/panels/map"
  - "internal/senses"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/senses-trail.json"
  - "internal/combat/fightend.go"
  - "internal/combat/fighter.go"
  - "internal/combat/fight.go"
  - "cmd/d2bot/senseshistory.go"
  - "cmd/d2bot/senseshistory_test.go"
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

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Trail points carry an activity code; events ring; additive payload keys; panel colours, glyphs, events filter, 30 min window and activity share line
- [ ] #2 Tests: Go ring/events/wiring, payload golden, node panel test; whole-repo gate
<!-- AC:END -->

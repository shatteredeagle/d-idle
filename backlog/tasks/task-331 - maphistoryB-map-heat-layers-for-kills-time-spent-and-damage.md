---
id: TASK-331
title: 'maphistoryB: map heat layers for kills, time spent and damage'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:30:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 223000
card: "maphistoryB"
owns:
  - "internal/watch/web/panels/map"
  - "internal/senses"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/history-heat.json"
  - "cmd/d2bot/senseshistory.go"
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
- [ ] #1 Per-area heat memory (kills, dwell, damage), reset with a new game, history.heat on /api/senses?hist=1&heat=1, panel heat layer with legend
<!-- AC:END -->

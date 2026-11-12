---
id: TASK-333
title: 'maphistoryC: fight cards and failure markers on the map'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:35:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 224000
card: "maphistoryC"
owns:
  - "internal/watch/web/panels/map"
  - "internal/senses"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/senses-failures.json"
  - "internal/nav/roam.go"
  - "internal/nav/nav.go"
  - "internal/nav/covview_test.go"
  - "internal/combat/fighter.go"
  - "internal/combat/fightend.go"
  - "internal/combat/reach.go"
  - "internal/combat/fight.go"
  - "internal/combat/castfirst.go"
  - "cmd/d2bot/senseshistory.go"
  - "cmd/d2bot/senseshistory_test.go"
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
- [ ] #1 Fight card on click of a combat segment or kill marker; failure markers (roam leg failed, mouth-gate hold, ignore window, target fallback, nav yield) from passive hooks
<!-- AC:END -->

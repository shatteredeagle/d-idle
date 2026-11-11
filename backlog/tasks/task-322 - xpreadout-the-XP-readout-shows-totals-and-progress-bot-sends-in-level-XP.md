---
id: TASK-322
title: 'xpreadout: the XP readout shows totals and progress (bot sends in-level XP)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:09:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 221000
card: "xpreadout"
owns:
  - "internal/watch/hero.go"
  - "internal/watch/bridge.go"
  - "internal/watch/web/panels/hero-detail"
  - "internal/watch/hub.go"
  - "internal/watch/mapview.go"
  - "internal/watch/sim.go"
  - "internal/watch/xpreadout_test.go"
  - "internal/watch/testdata/topband_dom.js"
  - "internal/watch/testdata/topband_dom_hero-detail_*.txt"
  - "cmd/d2bot/play.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the bot sends XP inside the level (updateCharacter) but /api/hero treated it as a total, so xp < level start and xpPct stayed null: the tile showed "-". Done when: the hub converts to totals (AbsoluteXP), the tile shows "XP cur / next (pct)" and xp/h.

result: fixed offline; Go + node DOM tests pass; not verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 /api/hero carries total xp, next level and percent from the runner status
- [x] #2 hero-detail tile shows XP current / needed (pct) and xp/h
- [x] #3 Go and node tests
<!-- AC:END -->

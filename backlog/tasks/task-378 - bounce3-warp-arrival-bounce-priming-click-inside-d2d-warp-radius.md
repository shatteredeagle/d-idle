---
id: TASK-378
title: 'bounce3: warp arrival bounce, priming click inside d2d warp radius'
status: In Progress
assignee:
  - pro-runner
created_date: '2026-11-12T20:03:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 241000
card: "bounce3"
owns:
  - "internal/d2dinput"
  - "internal/bot/loop.go"
  - "internal/bot/observe.go"
  - "internal/bot/areaskip_test.go"
  - "tools/bounce3"
  - "docs/bounce3.md"
exclusive: []
contracts: []
result: "Root cause V: the priming click after a level change landed within d2d's 2-cell warp-take radius of the return warp (walk_to/use_warp); kickSafe places it >= 3 cells from every warp; headless repro 6/12 bounces on main, 0/12 on the branch; breaker skips the deeper side; minute line bouncesPer100. Not yet verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the hero ping-ponged Cold Plains and Cave Level 1 (34 of 99 arrivals bounced on devp4) and the breaker banned Cold Plains. Done when: the headless repro passes, the breaker never bans the shallower side and the sandbox bounce rate is measured after deploy.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 End-to-end headless repro fails on main and passes on the branch (tools/bounce3/repro.sh)
- [x] #2 Priming click lands outside the warp take radius of every warp (unit tests)
- [x] #3 Bounce breaker skips the deeper side, never the transit (unit test)
- [x] #4 Minute line carries bouncesPer100
- [ ] #5 Bounce rate on the sandbox over 30 min after deploy measured (live)
<!-- AC:END -->

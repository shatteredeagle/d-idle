---
id: TASK-209
title: 'approachfast: short combat approach steps cost 2 s each'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T20:54:00Z'
updated_date: '2026-11-08T21:22:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: medium
ordinal: 162000
card: "approachfast"
owns:
  - "internal/nav"
  - "docs/walksmooth.md"
exclusive: []
contracts: []
result: "offline: GoTo polls position only (full read every 500 ms, at start, stall, level change); Paladin 5 to 1, Sorceress 8 to 2 full snapshots per step in the nav sim; dev container after the deploy: long approach legs observe 220 to 127 ms, short steps 218 to 167 ms (docs/walksmooth.md)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: short one-waypoint combat approach steps cost about 1.5 s, 0.6 s of it full snapshots (about 200 ms each). Done when: GoTo polls position only between full reads and the dev container shows shorter steps (docs/walksmooth.md, approachfast).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GoTo walks poll the position only between full snapshots (class-free, stuck detector on position)
- [x] #2 Tests: Paladin and Sorceress approach steps, stuck detector, race
- [x] #3 Before/after numbers from the dev container recorded in docs/walksmooth.md
<!-- AC:END -->

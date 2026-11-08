---
id: TASK-196
title: 'walksmooth2: Paladin still pauses between moves'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T16:57:00Z'
labels:
  - bot
  - bug
  - tester
milestone: m-4
dependencies: []
priority: high
ordinal: 157000
card: "walksmooth2"
result: "Measured live in the dev container (not a lab run): one observe took 850 ms because the walker, the safety guard and the minimap feed each queued full snapshots on the one-line-per-frame d2d channel. Minimap now reuses the walker snapshot and the guard reuses a snapshot younger than 350 ms (Reader.Recent); travel 3.6-5.9 -> 9.3-11.5 subtiles/s. Verified live in the container; offline tests added. Left: fork pump (one line per frame) and position-only polls (docs/walksmooth.md)."
owns:
  - "internal/nav/trace.go"
  - "internal/nav/walk.go"
  - "internal/d2dinput"
  - "internal/nav/nav.go"
  - "internal/nav/trace_test.go"
  - "internal/d2denv"
  - "internal/bot/observe.go"
  - "internal/bot/loop.go"
  - "internal/bot/guardreuse_test.go"
  - "cmd/d2bot/sensesunits.go"
  - "cmd/d2bot/sensesunits_test.go"
  - "cmd/d2bot/main.go"
  - "docs/walksmooth.md"
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
- [x] #1 Click-timing trace nav.trace in the dev image
- [x] #2 Measured causes of the remaining pauses on the live Paladin
- [x] #3 Fix with regression test
<!-- AC:END -->

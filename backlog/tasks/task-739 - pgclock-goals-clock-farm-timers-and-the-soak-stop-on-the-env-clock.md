---
id: TASK-739
title: 'pgclock: goals clock, farm timers and the soak stop on the env clock'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:47:00Z'
updated_date: '2026-11-27T18:49:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 345000
card: "pgclock"
owns:
  - "cmd/d2bot/d2dbackend.go"
  - "internal/bot/farm.go"
  - "internal/goals/clock_test.go"
  - "internal/bot/farmclock_test.go"
  - "internal/d2denv/reader.go"
  - "internal/d2denv/recent_test.go"
  - "internal/d2denv/env.go"
  - "cmd/d2bot/goalsclock_test.go"
  - "docs/d2d-determinism.md"
exclusive: []
contracts: []
result: "Bot-side wall-clock reads that move the command stream now run on the env clock: goals.Manager on the d2d path (d2dGoalsOptions, B1), the time-boxed farm timers (B2), and the Reader.Recent age check that decides whether the safety guard reuses the last snapshot or queues a fresh read (found by reading the journal pair of group 1: a 350 ms WALL window, systematic on a loaded builder; this is the read-only probe choice at equal state, S read, not yet proved by a builder pair). decide.Clock was already the env clock (B4, read). B3 done: the soak stop is d2denv.LateClock.StopAfter, run on the stepping goroutine after the first lockstep sleep that reaches the deadline (no second Pacer watcher needed); offline test TestStopAfterFiresOnTheSteppingGoroutine, -race; the builder pair with D2DSOAK_HOURS not re-run (U). Must-fail tests: farm resume duration (fails on the wall clock), goals options clock, Reader.Recent virtual age; two classes each. -race -count=20 on the touched packages. The determinism-run -n 3 pair on the builder: see docs/probes/pg-gate-group2-2026-11-21.md (not live-verified unless that doc says so)."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design 2.3 B1 to B3, 7 card 5. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 goals.Manager gets the env clock on the d2d path
- [x] #2 farm budget uses the env clock
- [x] #3 soak stop is tick based
- [x] #4 tests for two builds
<!-- AC:END -->

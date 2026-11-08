---
id: TASK-140
title: 'funwire: connect return screen, tally strip and Codex to the live bot'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T02:12:00Z'
labels:
  - bot
  - feature
  - G-C
milestone: m-4
dependencies:
  - TASK-127
  - TASK-129
  - TASK-130
priority: high
ordinal: 129000
card: "funwire"
owns:
  - "cmd/d2bot/funwire.go"
  - "cmd/d2bot/funwire_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "internal/watch/funwire.go"
  - "internal/watch/funwire_test.go"
  - "internal/watch/hub.go"
  - "internal/watch/http_answer.go"
  - "internal/watch/returnscreen.go"
  - "internal/watch/serve.go"
  - "internal/watch/tallystrip.go"
  - "internal/watch/web/*"
  - "internal/bot/runner.go"
  - "internal/bot/thoughts.go"
  - "internal/bot/waypoints.go"
  - "internal/bot/funhooks_test.go"
  - "docs/return-screen.md"
  - "docs/tally-strip.md"
  - "docs/codex.md"
  - "backlog/tasks/task-127*"
  - "backlog/tasks/task-129*"
  - "backlog/tasks/task-130*"
exclusive: []
contracts: []
result: "Wired and tested (watch e2e on a real server for two builds, cmd/d2bot feedHub test, bot hook tests); whole-repo gate green on the worker host. Live smoke 10 min on d2d: real events arrived, see docs/codex.md Wiring. Tally not live (needs the overlay)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Wiring job for the first playable slice of the fun loop (ADR 0018). Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 feedHub feeds the return screen, tally strip and Codex; routes mounted; charter sink on the order route
- [x] #2 End-to-end offline test (real server, two builds) passes
- [x] #3 Live smoke result recorded, or skipped with the reason
<!-- AC:END -->

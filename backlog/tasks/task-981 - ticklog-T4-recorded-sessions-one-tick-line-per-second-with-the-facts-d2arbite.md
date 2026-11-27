---
id: TASK-981
title: >-
  ticklog: T4 recorded sessions: one tick line per second with the facts,
  d2arbite
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-26T03:56:00Z'
updated_date: '2026-11-27T18:49:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: high
ordinal: 469000
card: "ticklog"
owns:
  - "docs/design/ticklog-t4.md"
  - "docs/arbiter.md"
  - "internal/bot/ticklog.go"
  - "internal/bot/ticklog_test.go"
  - "internal/bot/shadow.go"
  - "internal/env/tuning_bot.go"
  - "internal/arbiter/replay.go"
  - "internal/arbiter/replay_ticklog_test.go"
  - "data/darklaunch.json"
exclusive: []
contracts: []
result: "Tick line built behind bot.tickLogSec (default 0, dark-launch row), rate-limited by the bot clock, facts of the arbiter view, independent of the shadow; the replay (d2arbiter, -follow) reads tick lines and skips minute lines then. Offline tests only, NOT verified live: no sandbox night recorded yet (procedure in docs/design/ticklog-t4.md). Not build-specific. Remaining (live): the sandbox night recording and replay (AC 2). Remaining (live): AC 2 replay of one real sandbox night. Remaining (live): AC 2 replay of one real sandbox night with -follow documented."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Cause class killed: A before live; gives the arbiter migration an outcome measure. Roadmap row T4 (docs/research/fable-logic-review-2026-11-25.md 5.3, 7.3). One `tick` line per second with the facts the shadow adapter already collects (internal/bot/shadow.go), rate-limited, off by default; cmd/d2arbiter -follow replays them; makes the arbiter fixtures real (the Dark Wood fixture's facts are assumed, docs/arbiter.md). Risk: low (log volume). Enable criteria: tick log on both sandboxes for a night, replay runs without a panic.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tick line behind a knob with a ledger row
- [ ] #2 Replay of one real sandbox night with -follow documented
<!-- AC:END -->

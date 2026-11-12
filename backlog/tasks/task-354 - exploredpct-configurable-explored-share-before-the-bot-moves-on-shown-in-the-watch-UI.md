---
id: TASK-354
title: 'exploredpct: configurable explored share before the bot moves on, shown in the watch UI'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T03:08:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 231000
card: "exploredpct"
owns:
  - "docs/exploredpct.md"
  - "internal/bot/exploredpct.go"
  - "internal/bot/exploredpct_test.go"
  - "internal/watch/heroexplored_test.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "docs/knobs.md"
  - "internal/bot/goal.go"
  - "internal/bot/observe.go"
  - "internal/env/tuning_bot.go"
  - "internal/watch/hero.go"
  - "internal/watch/hub.go"
  - "internal/watch/web/panels/activity/panel.js"
exclusive: []
contracts: []
result: "bot.exploredPct (default 100, 30..100) ends a clear roam early via exploredEnough in sweepPass; /api/hero explored block, activity panel header, minute line; unit-tested, NOT verified live. Decision: nav roam() ErrExplored -> bot sweepPass cleared[area]."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 bot.exploredPct knob (default 100 = old behaviour) ends a clear roam early
- [x] #2 explored block in /api/hero and the activity panel header, minute log line
- [x] #3 tests and docs/exploredpct.md
<!-- AC:END -->

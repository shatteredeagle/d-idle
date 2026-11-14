---
id: TASK-398
title: 'staletarget: stale targetArea, border-loop ban, blocked step fallback'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T10:09:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 248000
card: "staletarget"
owns:
  - "internal/bot/staletarget.go"
  - "internal/bot/staletarget_test.go"
  - "internal/bot/loop.go"
  - "internal/bot/task.go"
  - "internal/bot/geargate.go"
  - "internal/bot/goal.go"
  - "internal/bot/whyline.go"
  - "internal/task/executor.go"
  - "internal/watch/whyline.json"
  - "internal/env/tuning_bot.go"
  - "docs/staletarget.md"
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
- [ ] #1 Stale targetArea no longer cancels quest steps or trips the border-loop breaker
- [ ] #2 Ban reason is real; non-death bans do not block waypoint or quest routes; refusal spin guard
- [ ] #3 Blocked journey step farms by level plan after a knob delay and retries with backoff
- [ ] #4 Regression tests for Paladin and Sorceress; whole-repo gate green
<!-- AC:END -->

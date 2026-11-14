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
  - "docs/knobs.md"
  - "internal/bot/areaskip_test.go"
  - "internal/bot/runner.go"
  - "internal/bot/observe.go"
  - "internal/bot/ordersguard.go"
  - "internal/goals/runner.go"
  - "internal/watch/whyline_test.go"
  - "internal/task/executor_unpark_test.go"
result: "Fixed offline, not deployed, no live run. Stale targetArea dropped on arrival from town and per quest tick; crossings stamped once per arrival (10 s gap); real ban reason in refusals; border-loop and non-death bans on waypoint-jumped hops never block the quest route; spin guard on refusals; blocked step farms the level plan after bot.questBlockedFarmMin (10) and retries with doubling wait; journeyBlocked status field. Tests: Sorceress and Paladin. Whole-repo gate on the worker host."
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
- [x] #1 Stale targetArea no longer cancels quest steps or trips the border-loop breaker
- [x] #2 Ban reason is real; non-death bans do not block waypoint or quest routes; refusal spin guard
- [x] #3 Blocked journey step farms by level plan after a knob delay and retries with backoff
- [x] #4 Regression tests for Paladin and Sorceress; whole-repo gate green
<!-- AC:END -->

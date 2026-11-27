---
id: TASK-511
title: 'farmloop: farm reset loop for leveling zones and zone clear dashboard settings'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-15T17:04:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 278000
card: "farmloop"
owns:
  - "internal/bot/farmreset.go"
  - "internal/bot/farmreset_test.go"
  - "internal/watch/zoneclear.go"
  - "internal/watch/zoneclear_test.go"
  - "internal/watch/web/panels/zoneclear"
  - "tools/levelsim/farmreset.go"
  - "tools/uishots/fixtures_zoneclear.go"
  - "docs/farmloop.md"
exclusive: []
contracts: []
result: "Farm loop built, shipped OFF (calibrated model +2.5 percent, prior +8, steep +16, never a loss with the value rule); /api/zoneclear and the opt-in zoneclear panel added; offline tests only, not verified live, fit-check fixtures not run (needs Chromium)"
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Farm loop in the bot (default off), value rule, guards, two-class tests
- [x] #2 levelsim -farmreset table and default choice
- [x] #3 GET/POST /api/zoneclear and the opt-in zoneclear panel with fit fixtures
- [ ] #4 Live check: 30 minutes in Cold Plains with bot.farmReset 1 (not run)
<!-- AC:END -->

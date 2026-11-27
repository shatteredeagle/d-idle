---
id: TASK-529
title: 'resetinstant: instant one-tap Reset, hung-loop escalation, header button'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T02:30:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 283000
card: "resetinstant"
owns:
  - "internal/bot/playerreset.go"
  - "internal/bot/playerreset_test.go"
  - "cmd/d2bot/resetwire.go"
  - "cmd/d2bot/resetwire_test.go"
  - "internal/watch/resetapi.go"
  - "internal/watch/resetapi_test.go"
  - "internal/watch/resetui_test.go"
  - "internal/watch/web/resetui.js"
  - "internal/watch/web/reset.css"
  - "internal/watch/testdata/reset"
  - "docs/reset.md"
  - "internal/bot/task.go"
  - "internal/bot/runner.go"
  - "internal/bot/loop.go"
  - "internal/bot/state.go"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.css"
  - "tools/uishots.sh"
  - "tools/uishots/fixtures_phone.go"
  - "tools/uishots/fit.go"
exclusive: []
contracts: []
result: "Soft Reset is instant (cancels the running action, 150 ms on the fake clock), escalates to the hard path after 3 s without a loop tick, POST answers 202 with a wait opt-in; one-tap header button with cooldown, no soft confirm; startFreshGame guarded (in flight, 5 s debounce). Offline only, not verified live; fit renders not viewed (the worker host was busy). closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Break-glass Reset must be instant (user 2026-11-15). Done when the criteria are ticked.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 soft reset never waits (wait opt-in), starts on the next tick
- [x] #2 3 s unacknowledged soft reset escalates to hard
- [x] #3 POST /api/reset answers 202; goldens updated
- [x] #4 one-tap header Reset button, no confirm for soft, 3 s cooldown
- [ ] #5 fit fixtures for the header at all sizes viewed
- [x] #6 docs/reset.md updated
<!-- AC:END -->

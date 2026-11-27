---
id: TASK-1069
title: 'logsink: bot operational log to stdout and a rotating file, d2bot ops log'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T21:54:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 515000
card: "logsink"
lane: next
owns:
  - "internal/logx"
  - "cmd/d2bot/main.go"
  - "cmd/d2bot/opscmd.go"
  - "cmd/d2bot/logsink_test.go"
  - "docs/ops"
exclusive: []
result: "root cause V: docker json-file log of both sandboxes is NUL-damaged (dockerd: Error decoding log file, invalid character x00), so docker logs stops at an old line while the bot still writes to the shared stdout pipe. Fix: the bot also writes <inst>/logs/bot.log (rotating) and d2bot ops log reads it. Offline tests only, not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The bot writes its log to stdout and to a rotating file (tests)
- [x] #2 d2bot ops log --tail N --grep reads it (tests)
- [x] #3 docs/ops/logs.md says where each log is and the verified root cause
- [ ] #4 LIVE: a sandbox on the next release shows d2bot ops log lines with inv= and tick
<!-- AC:END -->

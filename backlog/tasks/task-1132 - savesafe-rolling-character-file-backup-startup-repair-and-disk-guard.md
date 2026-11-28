---
id: TASK-1132
title: 'savesafe: rolling character file backup, startup repair and disk guard'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T09:49:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 537000
card: "savesafe"
lane: next
owns:
  - "internal/savesafe/"
  - "cmd/d2bot/savesafewire.go"
  - "cmd/d2bot/savesafewire_test.go"
  - "cmd/d2bot/opscmd.go"
  - "cmd/d2bot/opswire.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/ladder.go"
  - "internal/bot/savepoint.go"
  - "internal/bot/savepoint_test.go"
  - "internal/bot/runner.go"
  - "internal/bot/explain.go"
  - "internal/watch/notices.go"
  - "internal/watch/notices_test.go"
  - "internal/watch/serve.go"
  - "internal/watch/update.go"
  - "docs/design/fork-save-atomic.md"
  - "docs/ops/save-backups.md"
exclusive: []
contracts: []
result: "NOT verified live (unit tests with built character file images, Sorceress and Paladin). internal/savesafe copies a valid, changed d2s and its side files to state/savebackups every 10 wall minutes and at town arrival (temp, fsync, rename; newest 12 plus one a day for 7 days), restores the newest valid backup at startup before classForStart can read an empty file as no save, and refuses to write under 200 MB free. The planned save point is refused on a low disk (bot.Deps.DiskLow). Notices in ops status --why and a viewer bar; d2bot ops save check is read only. Nothing is build-specific: no skill, class or level rule. Not covered: recycle and NewGame paths still save on a full disk; only the managed d2d path is wired; the fork still writes in place (docs/design/fork-save-atomic.md)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Incident 2026-11-28: a full disk left the sandbox hero's d2s at 0 bytes. Done when the bot keeps rolling copies, repairs at startup and refuses to save on a full disk.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Rolling backup: valid d2s only, changed only, every 10 min and at town arrival, atomic, keep 12 plus one a day for 7 days
- [x] #2 Startup repair restores the newest valid backup over a missing, zero-length or invalid save, keeps the broken file, logs it
- [x] #3 Disk guard: nothing written and no planned save point under 200 MB, warning under 2 GB in ops status --why
- [x] #4 Repair and disk notices show in ops status --why and on the viewer
- [x] #5 d2bot ops save check lists the live validity and the backups, read only
- [x] #6 Fork request docs/design/fork-save-atomic.md and runbook docs/ops/save-backups.md
- [ ] #7 LIVE: on a sandbox, truncate a copy hero's d2s while stopped, start, see save repair in ops log and the viewer bar
<!-- AC:END -->

---
id: TASK-161
title: 'unlockorderqueue: order queue'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: low
ordinal: 139000
card: "unlockorderqueue"
owns:
  - "internal/watch/unlockorderqueue.go"
  - "internal/watch/unlockorderqueue_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: Den of Evil cleared once; queue 2 or 3 orders, one slot per town service reached. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: check Den of Evil clear conditions against the data; orders must add agency and not make the watcher redundant.
<!-- SECTION:DESCRIPTION:END -->

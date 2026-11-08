---
id: TASK-156
title: 'unlockquestlog: quest log'
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
priority: medium
ordinal: 134000
card: "unlockquestlog"
owns:
  - "internal/watch/unlockquestlog.go"
  - "internal/watch/unlockquestlog_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: Akara hands out the Den of Evil quest; list of quests with state. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: check Den of Evil quest state and reward conditions against the data before building.
<!-- SECTION:DESCRIPTION:END -->

---
id: TASK-163
title: 'unlockhall: Hall of Fame and replay'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - feature
  - parked
milestone: m-4
dependencies:
  - TASK-153
  - TASK-135
priority: low
ordinal: 141000
card: "unlockhall"
owns:
  - "internal/watch/unlockhall.go"
  - "internal/watch/unlockhall_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: retire after Andariel or first death; gallery of retired heroes with legacy charms and a 30 s replay clip on death. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: depends on legacyhall (TASK-135); opt-in retire stays ledger and UI only.
<!-- SECTION:DESCRIPTION:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: Hall waits for a live hero that reaches it; 0 boxes defined.
<!-- SECTION:NOTES:END -->

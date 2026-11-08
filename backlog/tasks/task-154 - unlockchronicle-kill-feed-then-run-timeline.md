---
id: TASK-154
title: 'unlockchronicle: kill feed, then run timeline'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:15:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: medium
ordinal: 132000
card: "unlockchronicle"
owns:
  - "internal/watch/unlockchronicle.go"
  - "internal/watch/unlockchronicle_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: first kill opens the kill feed; the first expedition card read opens the run timeline (M follow-up). Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: kills must come from observed events, the same source as the tally.
<!-- SECTION:DESCRIPTION:END -->

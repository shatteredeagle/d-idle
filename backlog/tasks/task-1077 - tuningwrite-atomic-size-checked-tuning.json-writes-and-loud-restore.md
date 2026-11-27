---
id: TASK-1077
title: 'tuningwrite: atomic, size-checked tuning.json writes and loud restore'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T23:22:00Z'
updated_date: '2026-11-27T23:47:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 517000
card: "tuningwrite"
lane: next
owns:
  - "internal/store/"
  - "internal/bot/tuner.go"
  - "internal/bot/tuner_test.go"
  - "tools/flip/"
exclusive: []
contracts: []
result: "store.writeAtomic refuses writes under 20 bytes, checks the temp size before rename, ENOSPC keeps the old file; Tuner logs one Warn with the restored knob count (restore from the existing .bak); flip.py verify and pre-flip fail hard on a zero-byte or missing file. Offline only, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 store write refuses sub-floor sizes, checks the temp size, ENOSPC keeps the old file
- [x] #2 load of a zero-byte file restores from .bak with one Warn naming the knob count
- [x] #3 flip.py verify fails hard on a zero-byte or missing tuning file
<!-- AC:END -->

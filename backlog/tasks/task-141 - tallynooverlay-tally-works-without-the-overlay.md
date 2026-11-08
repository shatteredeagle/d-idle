---
id: TASK-141
title: 'tallynooverlay: tally works without the overlay'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T02:44:00Z'
labels:
  - bot
  - bug
  - G-C
milestone: m-4
dependencies:
  - TASK-129
  - TASK-140
priority: high
ordinal: 130000
card: "tallynooverlay"
owns:
  - "internal/currency"
  - "internal/watch"
  - "cmd/d2bot"
  - "docs/tally-strip.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Live smoke on vanilla d2d showed the strip off. Done when the tally credits from events on vanilla d2d and game.exe.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Events-mode ledger credits the tally without the overlay; bank, marks and other currencies stay refused
- [x] #2 Strip says off only when the ladder or watch is off, with a reason
- [x] #3 Regression test (two classes, with and without overlay) fails before and passes after
<!-- AC:END -->
result: "Events-mode ledger; tally credits without the overlay (offline tests, two classes, with and without overlay). Not verified live."

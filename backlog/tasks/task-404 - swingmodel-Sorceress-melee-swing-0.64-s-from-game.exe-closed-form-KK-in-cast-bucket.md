---
id: TASK-404
title: 'swingmodel: Sorceress melee swing 0.64 s from game.exe closed form, KK in cast bucket'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T11:00:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 250000
card: "swingmodel"
result: "Sorceress swing 0.64 s (game.exe closed form, data table), Paladin 0.6 unchanged, mode 12 counted as cast; offline tests only, not verified live"
owns:
  - "internal/resource/swing.go"
  - "internal/resource/swing_test.go"
  - "internal/resource/sim.go"
  - "internal/bot/killtime.go"
  - "internal/bot/killtime_test.go"
  - "docs/swingmodel.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sorceress MeleeSec 0.64 and Paladin 0.6 from swing data, tested
- [x] #2 mode 12 counts as casting in killtime
- [x] #3 docs/swingmodel.md notes d2d not changed
<!-- AC:END -->

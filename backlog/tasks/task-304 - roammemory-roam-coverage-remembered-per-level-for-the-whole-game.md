---
id: TASK-304
title: 'roammemory: roam coverage remembered per level for the whole game'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T15:54:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 211000
card: "roammemory"
owns:
  - "internal/nav/roam.go"
  - "internal/nav/nav.go"
  - "internal/nav/stuckmem.go"
  - "internal/nav/roam_test.go"
  - "docs/roamspacing.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Per-level memory keyed by LevelMap identity; no persistence (bot restart = new d2d game, managed d2d). docs/roamspacing.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: re-entering a level restarted its roam from zero. Done when: per-level memory for the game.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 coverage survives A to B to A and is dropped on a new game
- [x] #2 overlay payload shows remembered cells on re-entry
<!-- AC:END -->

---
id: TASK-157
title: 'unlockidreveal: identify reveal'
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
ordinal: 135000
card: "unlockidreveal"
owns:
  - "internal/watch/unlockidreveal.go"
  - "internal/watch/unlockidreveal_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: card flip when the first magic or rare item is picked up; later affixes, colour, sound; first unique or set later. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: check how the bot handles unidentified items before the reveal; mods and uniques only after the first encounter.
<!-- SECTION:DESCRIPTION:END -->

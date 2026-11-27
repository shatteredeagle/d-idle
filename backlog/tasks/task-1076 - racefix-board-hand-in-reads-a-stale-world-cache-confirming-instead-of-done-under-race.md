---
id: TASK-1076
title: 'racefix: board hand-in reads a stale world cache (confirming instead of done under -race)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T22:57:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 516000
card: "racefix"
lane: next
owns:
  - "internal/bot/questboard.go"
  - "internal/bot/questboard_test.go"
exclusive: []
contracts: []
result: "Root cause: boardHandIn signals done while the loop still holds stepMu, so the tapper boardRT.world() TryLock fails and reads the pre-talk cached world (quest word not yet Save): chunk Done by the handed proxy, outcome confirming. Fix: drop and rebuild the cache inside boardHandIn. 300/300 under -race -cpu 1,2,4 (not verified live; baseline never reproduced the flake, 1 in 600 originally)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The hand-in tap rebuilds the world cache before the tapper reads the quest word
- [x] #2 TestQuestBoardAcceptRunStopHandIn passes 300/300 under -race -cpu 1,2,4
<!-- AC:END -->

---
id: TASK-115
title: 'qqueue: questions queue for the player''s return; remove the 45 s fade (D2)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
labels:
  - bot
  - feature
  - ladder
  - T0
milestone: m-4
dependencies:
  - TASK-19
priority: medium
ordinal: 108000
card: "qqueue"
owns:
  - "internal/decide/queue.go"
  - "internal/decide/queue_test.go"
  - "docs/contracts/decision-queue.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: user decision D2 (d.idle is an idle game), implementation is this later card. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Unanswered questions do not fade for the player: the hero acts on its default and the question queues grouped by kind for the return screen; the 45 s fade is removed as a player-facing rule
- [ ] #2 Test against two builds; go vet and go test -p 1 of internal/decide; docs say built vs verified live
- [ ] #3 Contract file is new only if the card owner decides; otherwise drop it from owns
<!-- AC:END -->

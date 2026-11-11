---
id: TASK-115
title: 'qqueue: questions queue for the player''s return; remove the 45 s fade (D2)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
updated_date: '2026-11-11T21:25:00Z'
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

Deferred 2026-11-07 (ADR 0018): folded into returnscreen (TASK-127, missed questions grouped, queued for the return) and proposezone (TASK-131, the hero waits and nothing fades for the player). Reopen only for the decide-side queue if those two need it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Unanswered questions do not fade for the player: the hero acts on its default and the question queues grouped by kind for the return screen; the 45 s fade is removed as a player-facing rule
- [ ] #2 Test against two builds; go vet and go test -p 1 of internal/decide; docs say built vs verified live
- [ ] #3 Contract file is new only if the card owner decides; otherwise drop it from owns
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed as superseded in the 2026-11-10 true-up: folded into the return screen by ADR 0018.

Driftfix 2026-11-11: reopened; Done had open offline boxes that no live-check row covers and the card result does not show them built.
<!-- SECTION:NOTES:END -->

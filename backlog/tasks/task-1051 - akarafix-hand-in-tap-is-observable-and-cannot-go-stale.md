---
id: TASK-1051
title: 'akarafix: hand-in tap is observable and cannot go stale'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T18:31:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 510000
card: "akarafix"
lane: next
owns:
  - "internal/bot/questboard.go"
  - "internal/bot/questboard_goal_test.go"
  - "internal/bot/goal.go"
  - "internal/questboard/desk.go"
  - "internal/questboard/desk_reason.go"
  - "internal/questboard/sm_reason.go"
  - "internal/questboard/desk_test.go"
  - "cmd/d2bot/questwire.go"
  - "docs/questboard.md"
result: "Tap logs queued/served/dropped; stale or timed-out requests leave the mailbox; handin_waiting reason in the API; idle goal serves the tap in town. Offline tests only, not verified live; live drill open."
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
- [x] #1 Log lines when the hand-in tap is queued, served and dropped
- [x] #2 A cancelled or timed-out request is dropped from rt.reqs (test fails on the old code)
- [x] #3 HandInView.why / chunk reason handin_waiting says why the tap waits (API only)
- [x] #4 Idle goal serves the hand-in in town (runs goal: same call, not driven in a test)
- [ ] #5 Live drill on a lab slot (Proving Ground cain.read, writes after start: 0)
<!-- AC:END -->

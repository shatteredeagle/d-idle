---
id: TASK-1061
title: 'diffgo: player-facing go for the difficulty hop (ops verb, persisted go, Return button)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T20:00:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 514000
card: "diffgo"
lane: next
owns:
  - "internal/diffready/consent.go"
  - "internal/bot/diffready.go"
  - "internal/bot/diffready_test.go"
  - "internal/bot/explain.go"
  - "internal/ops"
  - "cmd/d2bot/opscmd.go"
  - "cmd/d2bot/opswire.go"
  - "cmd/d2bot/funwire.go"
  - "cmd/d2bot/live.go"
  - "internal/watch/difficultydesk.go"
  - "internal/watch/difficultydesk_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/returnscreen.go"
  - "docs/diffready.md"
exclusive: []
contracts: []
result: "Built and tested offline, not verified live: verb go-difficulty, persisted go, Return screen button, explain difficultyOffer."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 go-difficulty verb refuses when not ready, consumed once, kept over restart (tests)
- [x] #2 Return screen button and POST /api/difficulty/go; explain JSON difficultyOffer
<!-- AC:END -->

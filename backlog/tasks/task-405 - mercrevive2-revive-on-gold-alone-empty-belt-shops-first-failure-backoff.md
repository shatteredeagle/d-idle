---
id: TASK-405
title: 'mercrevive2: revive on gold alone, empty belt shops first, failure backoff'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T11:01:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 250000
card: mercrevive2
owns:
  - "docs/mercrevive.md"
  - "internal/merc"
  - "cmd/d2bot/mercwire.go"
  - "cmd/d2bot/mercwire_test.go"
  - "docs/contracts/merc-policy.md"
exclusive: []
contracts:
  - merc-policy
result: "ShouldRevive on gold only; empty belt shops first; failed revive backs off 2 min; contract line reworded; B4 proposal only. Offline tests, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the game revives on gold alone (V read, merc2-trace); the bot also demanded a portal and potions. Done when: see docs/mercrevive.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ShouldRevive needs gold only
- [x] #2 empty belt shops first, failed revive backs off
<!-- AC:END -->

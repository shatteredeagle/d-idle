---
id: TASK-1060
title: 'advisorwire: bot-side stock and Return screen advice (text only)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T19:55:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 512000
card: "advisorwire"
lane: next
owns:
  - "cmd/d2bot/advisewire.go"
  - "cmd/d2bot/advisewire_test.go"
  - "cmd/d2bot/funwire.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
exclusive: []
contracts: []
result: "Offline only, not verified live: cmd/d2bot/advisewire.go builds advisor.Stock from hero pack, cube, stash and vault, reads items.advisor, sets FunConfig.Advice, cached by stock fingerprint, text only. Tests: Necromancer, Paladin, knob 0, cache. Vault page strip skipped (no advice plumbing there). Filled sockets not read (U)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Return screen shows 'Worth making' from hero pack, stash and vault when items.advisor is 1; nothing at 0
- [x] #2 Text only: no item is moved, cached by inventory fingerprint
<!-- AC:END -->

---
id: TASK-149
title: 'sensesunlock: unlock ladder from Codex firsts, level and tally spend'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:12:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-148
  - TASK-140
  - TASK-141
priority: low
ordinal: 132000
card: "sensesunlock"
owns:
  - "internal/watch/sensesunlock.go"
  - "internal/watch/sensesunlock_test.go"
  - "internal/watch/senses_tiers.json"
  - "internal/watch/senses.go"
  - "internal/watch/testdata/senses-ladder-*.json"
  - "data/unlocks/ladder.json"
  - "docs/senses-unlock.md"
exclusive: []
contracts: []
result: "Seven senses rows in data/unlocks/ladder.json, watch.LadderUnlocker, /api/senses gating with locked list and next, golden JSON tiers 0-4, buying tiers 5-6; offline only (go test ./internal/watch ./internal/unlock), not verified live. AC4 open: registry rows not owned here. cmd/d2bot still mounts FixedTier."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md. Not on the two-week playable path (returnscreen, policysheet, tallystrip, codexfirsts, playsession2); after it. Needs codexfirsts wiring (TASK-130, TASK-140) and tally (TASK-129, TASK-141). Done when the ladder unlocks from a synthesized Codex and ledger run.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tier table as data (tier, sense, trigger kind: codex/level/spend, trigger value); persisted per profile beside the Codex file; never relocks
- [x] #2 /api/senses returns only unlocked tiers (locked keys absent), plus a teaser for the next tier
- [x] #3 Return screen shows a New sense notice; the unlock raises no question
- [ ] #4 Tier 5/6 spend rows registered in the tally ledger registry (with the tallystrip owner)
<!-- AC:END -->

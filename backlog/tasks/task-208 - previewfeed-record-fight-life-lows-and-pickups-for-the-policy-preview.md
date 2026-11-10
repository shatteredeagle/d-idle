---
id: TASK-208
title: 'previewfeed: record fight life lows and pickups for the policy preview'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T20:34:00Z'
labels:
  - bot
  - feature
milestone: m-4
dependencies: []
priority: medium
ordinal: 161000
card: "previewfeed"
owns:
  - "internal/watch/awayspool.go"
  - "internal/watch/awayspool_test.go"
  - "cmd/d2bot/previewfeed.go"
  - "cmd/d2bot/previewfeed_test.go"
  - "internal/combat/fightend.go"
  - "internal/combat/fightend_test.go"
  - "internal/combat/fight.go"
  - "internal/combat/fighter.go"
  - "internal/watch/returnscreen.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "docs/policy-preview.md"
exclusive: []
contracts: []
result: "Built and tested on fixtures only (go test combat, watch, cmd/d2bot); not run against a live bot. Pickups already had an emitter (goals drop events into the away log). Build-specific: none, the feed is class-agnostic; the preview reads class knobs from the sheet."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: the stance preview reads recorded fight life lows but nothing emitted them. Done when: every finished fight (3 s or damage) is one close event, persisted across restarts, and the preview returns deltas end to end.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Each finished fight emits one close event (lowest life percent) into the away log; quick unhurt fights are skipped
- [x] #2 Close and drop events survive a restart (spool)
- [x] #3 End-to-end fixture test: fake fights give real stance deltas for Paladin and Sorceress
- [x] #4 docs/policy-preview.md gap closed
<!-- AC:END -->

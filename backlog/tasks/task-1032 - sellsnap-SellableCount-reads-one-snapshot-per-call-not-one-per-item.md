---
id: TASK-1032
title: 'sellsnap: SellableCount reads one snapshot per call, not one per item'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T04:34:00Z'
updated_date: '2026-11-27T05:29:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 499000
card: "sellsnap"
lane: next
owns:
  - "internal/town/shop.go"
  - "internal/town/evalsnap.go"
  - "internal/town/sellsnap_test.go"
  - "internal/bot/sellable.go"
  - "internal/bot/resource.go"
  - "internal/bot/routeguard.go"
  - "internal/bot/runner.go"
  - "internal/bot/sellsnap_test.go"
exclusive: []
contracts: []
result: "SellableCount now judges the pack against the snapshot it is given (6 items: 6 snapshots before, 0 after; benchmark 10 to 0 per call); the bot judges once per snapshot and the guard world behind the readiness, risk and propose pages reads the cached count. Build-neutral, tested offline with the sim and fake rigs, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: a goroutine dump of the sandbox bot (2026-11-27) showed about 8 goroutines queued on the d2d client mutex: SellableCount judged every pack item with EvaluateUnit, which took a full world snapshot per item, and the dashboard, readiness check and guard all asked for it. Done when the count reads no snapshot and the watch API never runs the sell judgement.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 SellableCount takes no game snapshot (the town judges every pack item against the snapshot it is given); a must-fail test counts snapshot calls on a fake source
- [x] #2 The bot computes the sell count once per snapshot (memo), and the guard world used by the HTTP readiness, risk and propose pages reads only that cached value
- [x] #3 Benchmark for SellableCount; no wall-clock ceiling in the unit tests
<!-- AC:END -->

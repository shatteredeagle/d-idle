---
id: TASK-1032
title: 'sellsnap: SellableCount reads one snapshot per call, not one per item'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T04:34:00Z'
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SellableCount takes no game snapshot (the town judges every pack item against the snapshot it is given); a must-fail test counts snapshot calls on a fake source
- [ ] #2 The bot computes the sell count once per snapshot (memo), and the guard world used by the HTTP readiness, risk and propose pages reads only that cached value
- [ ] #3 Benchmark for SellableCount; no wall-clock ceiling in the unit tests
<!-- AC:END -->

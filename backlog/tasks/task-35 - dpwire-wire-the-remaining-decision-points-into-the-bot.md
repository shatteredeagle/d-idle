---
id: TASK-35
title: 'dpwire: wire the remaining decision points into the bot'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-06T16:02:00Z'
updated_date: '2026-11-07T21:56:00Z'
labels:
  - bot
  - feature
  - ladder
  - T0
  - parked
dependencies: []
priority: high
ordinal: 35000
card: "dpwire"
owns:
  - "docs/dp-wiring.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

docs/t0-slice.md says nine points are wired (card text exists for the rest, no call sites). Count of the whole catalogue: see internal/decide (23 points in docs/research/decision-point-store-notes.md; the brief for this task said 22: reconcile first). The coordinator adds the call-site files to owns at launch.

Done when: docs/dp-wiring.md lists every unwired point with its legacy default, the call site to hook, the parity...; Each point is wired behind ladder.Active with the legacy verdict as the untrained answer; a golden parity...; go vet and go test -p 1 of the changed packages; leakcheck 0; says verified live or not (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/dp-wiring.md lists every unwired point with its legacy default, the call site to hook, the parity witness test, and V or U tags
- [ ] #2 Each point is wired behind ladder.Active with the legacy verdict as the untrained answer; a golden parity test per point on two builds; refused points keep a witness
- [ ] #3 go vet and go test -p 1 of the changed packages; leakcheck 0; says verified live or not
<!-- AC:END -->

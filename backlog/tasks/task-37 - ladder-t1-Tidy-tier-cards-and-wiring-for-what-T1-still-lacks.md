---
id: TASK-37
title: 'ladder-t1: Tidy tier: cards and wiring for what T1 still lacks'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T16:02:00Z'
labels:
  - bot
  - feature
  - ladder
  - T1
dependencies: []
priority: medium
ordinal: 37000
card: "ladder-t1"
owns:
  - "docs/ladder-t1.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

ADR 0016 tier T1 Tidy. docs/t0-slice.md: not wired and no witness test: t1.identify, t1.keep.kind. Wired: t1.repair, t1.tp.lost.

Done when: docs/ladder-t1.md: every T1 point with default, card text (what happened, what each answer does, the...; t1.identify and t1.keep.kind wired behind ladder.Active with the untrained default equal to today's...; go vet and go test -p 1 of changed packages; leakcheck 0; says verified live or not (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/ladder-t1.md: every T1 point with default, card text (what happened, what each answer does, the default), call site and parity witness; tags V or U
- [ ] #2 t1.identify and t1.keep.kind wired behind ladder.Active with the untrained default equal to today's behaviour, tests on two builds
- [ ] #3 go vet and go test -p 1 of changed packages; leakcheck 0; says verified live or not
<!-- AC:END -->

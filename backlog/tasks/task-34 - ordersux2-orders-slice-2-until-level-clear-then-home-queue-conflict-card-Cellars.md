---
id: TASK-34
title: 'ordersux2: orders slice 2: until-level, clear then home, queue, conflict'
status: Proposed
assignee: []
created_date: '2026-11-06T16:02:00Z'
labels:
  - T0
  - watch
dependencies: []
priority: medium
ordinal: 34000
card: "ordersux2"
owns:
  - "internal/watch/orders.go"
  - "internal/watch/mapview.go"
  - "internal/watch/ordersmap_test.go"
  - "docs/orders-slice2.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 2 of docs/research/orders-ux-design.md section 11 ('Left for slice 2'). Slice 1 is merged and offline tested only.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Until level N: the Hub ends the order on a clvl poll (POST /api/orders until=level is no longer refused); test with a fake clock and two builds
- [ ] #2 Clear then come home is a farm-then-home pair of goals; the order sheet offers it as a chip; test for the pair and for cancel mid-way
- [ ] #3 Add to queue and pos back on POST /api/orders; the Up next list is writable; tests for order of start
- [ ] #4 Conflict Thought card: an order the guard refuses raises the t0.area.toohard question with the guard text; a learned go lifts it; tests; the Forgotten Tower Cellars is an orderable map node
- [ ] #5 Docs/orders-slice2.md says built vs verified live; go vet and go test -p 1 of internal/watch; leakcheck 0
<!-- AC:END -->

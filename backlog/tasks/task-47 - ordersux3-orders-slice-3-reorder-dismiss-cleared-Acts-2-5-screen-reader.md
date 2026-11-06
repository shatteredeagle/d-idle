---
id: TASK-47
title: 'ordersux3: orders slice 3: reorder, dismiss, cleared, Acts 2-5, screen reader'
status: Proposed
assignee: []
created_date: '2026-11-06T16:03:00Z'
updated_date: '2026-11-06T16:03:00Z'
labels:
  - T0
  - watch
dependencies:
  - TASK-34
priority: low
ordinal: 47000
card: "ordersux3"
owns:
  - "internal/watch/mapdata.go"
  - "internal/watch/web/**"
  - "docs/orders-slice3.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 3 of docs/research/orders-ux-design.md. Starts after ordersux2 (shared file internal/watch/orders.go).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Reorder: POST /api/orders/move {id, delta}; failed orders keep strikes and can be dismissed; tests
- [ ] #2 Cleared marks come from a nav explored signal, not only quest events; test with a recorded map; the EvQuest marks stay as fallback
- [ ] #3 Acts 2 to 5 tables in mapdata.go, same schema (data only, orderable false until the bot can walk there); two-build test of the schema
- [ ] #4 Screen-reader pass of the orders page (roles, labels, focus order, live regions) with a written checklist in docs/orders-slice3.md; built vs verified live stated
<!-- AC:END -->

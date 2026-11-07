---
id: TASK-60
title: 'custom-item-1: stackable currency token from one early source'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-07T03:01:00Z'
updated_date: '2026-11-07T03:13:00Z'
labels:
  - d2d
  - feature
  - d2d-items
  - d2d-loot
  - metacurrency
  - T5
milestone: m-3
dependencies: []
priority: medium
ordinal: 58000
card: "custom-item-1"
owns:
  - "docs/research/custom-item-1.md"
exclusive: []
contracts: []
result: "Token row, TC line and bank row exist as overlay data (see TASK-63 and docs/research/currency-rows.md); the kill drill and the ledger credit are not done: live run pending (docs/probes/currency1.md) and internal/currency (job A) is not on main."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

The first currency token proves the whole custom-item path from one early source.

Done when: Token row, TC line and cube bank recipe in the overlay; offline drop-probability test passes; Drill kills N, pack count within the binomial interval, ledger credit equals pack loss (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Token row, TC line and cube bank recipe in the overlay; offline drop-probability test passes
- [ ] #2 Drill kills N, pack count within the binomial interval, ledger credit equals pack loss
<!-- AC:END -->

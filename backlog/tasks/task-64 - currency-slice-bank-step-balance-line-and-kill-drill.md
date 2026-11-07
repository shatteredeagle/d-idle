---
id: TASK-64
title: 'currency-slice: bank step, balance line and kill drill'
status: Proposed
assignee: []
created_date: '2026-11-07T03:13:00Z'
labels:
  - T5
dependencies: []
ordinal: 62000
card: "currency-slice"
owns:
  - "internal/currencyflow/"
  - "docs/probes/currency-drill.md"
exclusive: []
contracts: []
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Town bank step transmutes the stack and commits the credit; watch state carries a currency block
- [ ] #2 Drill kills N in d.idle mode: pack count in binomial interval, credit equals loss, writes after start 0
<!-- AC:END -->

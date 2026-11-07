---
id: TASK-62
title: 'currency-ledger: registry and account ledger (Go, offline)'
status: Proposed
assignee: []
created_date: '2026-11-07T03:13:00Z'
labels:
  - T5
dependencies: []
ordinal: 60000
card: "currency-ledger"
owns:
  - "internal/currency/"
  - "docs/currency.md"
exclusive: []
contracts: []
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Registry loads from data; hash-chained append-only ledger with write-ahead bank entries; tests on two currencies
- [ ] #2 Pack observer credits observed loss only, refuses cheated stacks, idempotent on replay
<!-- AC:END -->

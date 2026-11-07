---
id: TASK-62
title: 'currency-ledger: registry and account ledger (Go, offline)'
status: In Progress
assignee: []
created_date: '2026-11-07T03:13:00Z'
updated_date: '2026-11-07T04:13:00Z'
labels:
  - T5
dependencies: []
ordinal: 60000
card: "currency-ledger"
owns:
  - "internal/currency/"
  - "docs/currency.md"
  - "internal/watch/currency.go"
  - "internal/watch/currency_test.go"
  - "internal/watch/testdata/currency-*.json"
  - "cmd/d2bot/ladder_currency_test.go"
exclusive: []
contracts: []
result: "Built offline, NOT verified live. internal/currency: registry (JSON, validated), hash-chained flock ledger with head file, write-ahead bank (Begin/Commit/Recover), Observer interface and Fake, soft cap, spend; watch GET /api/currency, header strip and Spend tab shell (nothing wired); d2bot wallet only with --overlay-dir. Tests: go test -race ./internal/currency ./internal/watch ./cmd/d2bot. Page not seen in a browser. Format and operations: docs/currency.md."
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Registry loads from data; hash-chained append-only ledger with write-ahead bank entries; tests on two currencies
- [x] #2 Pack observer credits observed loss only, refuses cheated stacks, idempotent on replay
<!-- AC:END -->

Notes 2026-11-06 (job A of the metacurrency slice): owns widened with the watch currency files (new files internal/watch/currency*.go and golden files; TASK-64 owns only internal/currencyflow, so no overlap) plus small edits to the shared files internal/watch/{hub.go,views.go,http_view.go,web/*} and cmd/d2bot/{ladder.go,d2dbackend.go} (named in the commit; other open watch tasks touch web/* and orders files, merge conflicts are mechanical). Wiring for job B: `ladderRig.Wallet` (SetCarried, Ledger.Bank with an Observer).

## Why

Gate 1 of the d2d gate ladder was run on several seeds (16,17,18,19,20) unattended by tools/gate-run.sh (ops repo, docs/gate-run.md): a single seed gives no XP/h verdict, five do. This change carries the multi-seed record and the per-seed records.

## What Changes

- Seeds 16,17,18,19,20 judged together against the Gate 1 requirements of the soak-gates spec: NOT PASSED.
- Docs only: docs/soaks/gate1-multiseed-20261103-1226-ms1.md, one record and one compressed minutes log per seed. No bot or spec change. Fix cards for failing criteria are written by the reviewing agent.

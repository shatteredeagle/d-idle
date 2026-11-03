## Why

Gate 1 of the d2d gate ladder was run unattended by tools/gate-run.sh (ops repo, docs/gate-run.md): no agent polled it. This change carries the generated record.

## What Changes

- One hidden-d2d soak judged against the Gate 1 requirements of the soak-gates spec: NOT PASSED (True XP/h at or above the d2d baseline; Belt).
- Docs only: docs/soaks/gate1-20261103-0338.md and the compressed minutes log. No bot or spec change. Fix cards for failing criteria are written by the reviewing agent.

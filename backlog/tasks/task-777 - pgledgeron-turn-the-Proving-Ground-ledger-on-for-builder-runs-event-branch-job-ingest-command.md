---
id: TASK-777
title: >-
  pgledgeron: turn the Proving Ground ledger on for builder runs (event, branch,
  job, ingest command)
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T14:50:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - ops
  - feature
dependencies: []
priority: medium
ordinal: 359000
card: "pgledgeron"
owns:
  - "docs/process/pg-ledger-ops.md"
  - "tools/provingground/run.sh"
  - "tools/provingground/pgenv.sh"
  - "tools/provingground/pgenv_test.sh"
exclusive: []
contracts: []
result: "Builder runs export PG_EVENT, PG_BRANCH (from the worktree name) and PG_JOB and read PG_LEDGER_CMD from a host env file (pgenv.sh, sourced by run.sh; offline test pgenv_test.sh). Ledger lines reach the board host by a 10-minute PULL of the builders spools (no builder credential exists for a push), idempotent, ops test passes. NOT live-verified: no scenario has run since the wiring and the builder was unreachable; the status page is not regenerated (no lines). First-run checklist in docs/process/pg-ledger-ops.md."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Rollout step of the Proving Ground plan (TASK-743 code merged in group 1). Consumer: docs/process/pg-ledger-ops.md. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md section 5.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 builder runs export PG_EVENT, PG_BRANCH, PG_JOB and read the ingest command from host config
- [ ] #2 ledger lines reach the board host (ingest by pull from the spool, idempotent)
- [ ] #3 status page regenerated from the ledger
<!-- AC:END -->

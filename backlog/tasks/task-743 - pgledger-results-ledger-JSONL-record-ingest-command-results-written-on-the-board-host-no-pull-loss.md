---
id: TASK-743
title: >-
  pgledger: results ledger: JSONL record, ingest command, results written on the
  board host (no pull loss)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:48:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 349000
card: "pgledger"
owns:
  - "cmd/d2scenario/ledger.go"
  - "internal/scenario/ledger.go"
  - "docs/process/pg-ledger-ops.md"
  - "internal/scenario/ledger_test.go"
  - "cmd/d2scenario/ledger_test.go"
  - "cmd/d2scenario/run.go"
  - "cmd/d2scenario/main.go"
exclusive: []
contracts: []
result: "Ledger line (internal/scenario/ledger.go, design 5.1: run id, verdict, D1, D2, host role, bot commit, d2d hash, setup tag EARNED/SETUP/CHEATED/IMPOSSIBLE, event, branch) is written by exec-one into <out>/ledger.jsonl on the host that ran it and shipped through PG_LEDGER_CMD (ingest command on the board host, an ops-repo tool, described by role); a line without provenance or setup tag is refused. d2scenario ledger status-doc|check renders the generated status page. Nothing is EARNED yet (no earned-save card), runs BLOCKED before exec-one (setup not buildable) are not ledgered. Offline tests -race -count=20; ingest side verified offline only."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design section 5, 7 card 9 (Q6: ledger on the board host next to the status files, one generated status doc in the repo). Ops-repo part is a commit on a branch of the ops repo. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Record per design 5.1: run id, verdict, D1, D2, host role, commit, setup tag EARNED/SETUP/CHEATED/IMPOSSIBLE, history
- [x] #2 runner appends at the end of exec-one through one ingest command; append-only
- [x] #3 ingest and board-host side are an ops-repo change, described by role
- [x] #4 must-fail test: a run without provenance is refused
<!-- AC:END -->

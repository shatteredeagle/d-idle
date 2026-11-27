---
id: TASK-737
title: >-
  pgjournal: runner passes the journal variable; determinism-run -n N; the
  journal pair proof with d2ddet
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:46:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - probe
dependencies: []
priority: medium
ordinal: 343000
card: "pgjournal"
owns:
  - "cmd/d2scenario/journal.go"
  - "docs/probes/pg-journal-pair-2026-11-21.md"
  - "docs/probes/pg-gate-group1-2026-11-21.md"
  - "cmd/d2scenario/detrun.go"
  - "cmd/d2scenario/detrun_test.go"
  - "tools/provingground/run.sh"
exclusive: []
contracts: []
result: "determinism-run -n N and run -journal added (runner passes D2DCTL_JOURNAL, results carry journal, d2ddet built by run.sh and run on each pair). Pair run on the builder long lane, job 20261121-002134-7785: town.resupply and act1.burial.blood_raven, Paladin 11, three repetitions each: verdict, assertion digest and state digest equal in every pair; journals identical for about 28.8k and 32.6k records, then a bot-side read-only probe choice differs at equal state (tick 2242 and 7838), cause not pinned (U). CHEATED/SETUP hero, mechanics only. Record docs/probes/pg-gate-group1-2026-11-21.md."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design section 2.5, 7 card 3. About 40 builder minutes. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Runner passes D2DCTL_JOURNAL for the bot
- [x] #2 determinism-run -n N writes N runs and compares them
- [x] #3 Pair of design 2.5 (town.resupply and act1.burial.blood_raven, Paladin 11, 3 runs each) run on the builder main lane when free and written up with d2ddet compare output
<!-- AC:END -->

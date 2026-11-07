---
id: TASK-65
title: 'g2g-record: Gate 2 g2g wave 1 result'
status: In Progress
assignee: []
created_date: '2026-11-07T06:52:00Z'
labels:
  - gate
dependencies: []
ordinal: 63000
card: "g2grecord"
owns:
  - "docs/soaks/gate2-multiseed-20261107-0150-g2g.md"
  - "docs/research/g2g-diagnosis.md"
exclusive: []
contracts: []
result: "Gate 2 g2g wave 1 (seeds 11, 13, 17) NOT PASSED, indicative at 3 of 5 seeds: seed 11 PASS, seeds 13 and 17 fail NOKILL, seed 17 also funded BELT; record read from logs, nothing run live by this task. Diagnosis of the failures: docs/research/g2g-diagnosis.md (read from the logs and the worker-host bot logs, nothing run live)."
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Record built by gate-run.sh --collect-multi --gate 2 --tag g2g (3 seeds, indicative)
- [x] #2 Review notes: per-seed table, merc revive, mouthRef, early-death logs, tp=0 caveat
- [x] #3 Checks clean (leakcheck, 256 KB guard, contracts, backlog)
<!-- AC:END -->

---
id: TASK-40
title: 'tpreserve-live: prove the scroll reserve and no-portal leave with the tp field'
status: Proposed
assignee: []
created_date: '2026-11-06T16:02:00Z'
labels:
  - bot-defect
dependencies: []
priority: medium
ordinal: 40000
card: "tpreserve-live"
owns:
  - "docs/probes/tpreserve.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
g2efix F2b (scroll reserve town.tpReserve, minute-line field tp) was a hypothesis: docs/research/g2e-diagnosis.md says the cause was U because the log had no scroll count. The tp field is on main (agent tpfield).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Read the tp field in the next gate or soak minute lines; table of tp over time and the trips triggered by the reserve (tsc)
- [ ] #2 Verdict: the no-portal at low clvl does or does not recur; F2b confirmed or refuted with the log lines; tagged V or U
<!-- AC:END -->

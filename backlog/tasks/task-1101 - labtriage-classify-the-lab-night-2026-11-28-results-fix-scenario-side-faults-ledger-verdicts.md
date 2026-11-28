---
id: TASK-1101
title: 'labtriage: classify the lab night 2026-11-28 results, fix scenario-side faults, ledger verdicts'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:18:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 525000
card: "labtriage"
lane: next
owns:
  - "docs/probes/lab-night-results-2026-11-28.md"
  - "tools/provingground/scenarios/combatdrills.json"
  - "tools/provingground/scenarios/townplan.json"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Triage of the three Proving Ground jobs of the lab night. Done when the results doc is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every non-GREEN row of the three jobs is classified with log evidence
- [ ] #2 Scenario-side faults fixed, d2scenario validate ok
- [ ] #3 Cards written for the product bugs
- [ ] #4 Dark-ledger verdicts per row with the evidence
<!-- AC:END -->

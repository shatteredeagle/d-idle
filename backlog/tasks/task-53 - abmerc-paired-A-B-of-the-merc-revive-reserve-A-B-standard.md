---
id: TASK-53
title: 'abmerc: paired A/B of the merc revive reserve; A/B standard'
status: In Progress
assignee: []
created_date: '2026-11-06T22:36:00Z'
labels:
  - tooling
dependencies: []
ordinal: 52000
card: "abmerc"
owns:
  - "tools/ab/prep"
  - "tools/ab/mercrun.sh"
  - "tools/ab/mercab.sh"
  - "tools/ab/mercan.py"
  - "tools/ab/test_mercan.py"
  - "docs/ab-merc.md"
  - "docs/ab-design.md"
  - "docs/backlog-howto.md"
exclusive: []
contracts: []
result: "harness and one dry seed pair (0.5 virtual hour, slot 4) verified live on the worker; the full 16-run A/B (criterion 3, second half) not run yet, host busy; prep experience line untested live Status sync 2026-11-07: stays open, AC 3 (the full 16-run A/B) has not run."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
User decision 2026-11-06: set up the merc reserve A/B and make paired A/B the standard. Arm A = SME rule (merc.reviveReserveLow 0, High 50, Clvl 13); arm B = old policy emulated with fixed reserves.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Harness (prep tool, one-run script, driver, analysis with tests) for the clvl 11 start variant
- [x] #2 Dry run of one seed pair at a short horizon proves the harness end to end
- [ ] #3 Run plan (wall hours, slot hours, earliest start) written; the full 16-run A/B recorded when the host allows
- [x] #4 Rule in docs/backlog-howto.md and docs/ab-design.md: strategy changes get a paired A/B before the long gate
<!-- AC:END -->

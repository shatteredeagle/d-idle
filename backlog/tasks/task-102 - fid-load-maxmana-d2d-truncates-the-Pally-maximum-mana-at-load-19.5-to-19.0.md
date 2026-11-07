---
id: TASK-102
title: 'fid-load-maxmana: d2d truncates the Pally maximum mana at load (19.5 to 19.0)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:49:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-data
  - fidelity
milestone: m-2
dependencies:
  - TASK-55
priority: medium
ordinal: 100000
card: "fid-load-maxmana"
owns:
  - "docs/research/fid-loadmaxmana-trace.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Matrix row ST01. Evidence: docs/research/fidelity-probe-run1.md (Pally max mana file and game.exe 19.5 raw 4992, d2d 19.0 raw 4864, 3 of 3). Cause not found yet (the pilot trace lists the candidate sites). Needs a Paladin clvl 4 d2s as test input (the run-1 copy is on the worker host cold dir). Done when the cause is read in source and a test shows 4992 after the fix.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cause found in d2d (load path or Fight::update_fighters item_max/max share) with file:line
- [ ] #2 Reproduced in a d2d unit test with a Paladin clvl 4 stat block (raw 4992)
- [ ] #3 Fold into the pilot PR or a second PR; draft only
<!-- AC:END -->

---
id: TASK-330
title: 'zealdiff: d2d side of the Zeal strike spacing differential'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:21:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - fidelity
dependencies: []
priority: medium
ordinal: 224000
card: "zealdiff"
owns:
  - "tools/emu/zd_*"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fid-zeal.md: game.exe restarts the swing after every Zeal strike (Param2 100 through FUN_00553b10), d2d waits a full swing (V read); the d2d side was not run. Done when: the differential has mismatches before and zero after.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 drive d2d Fight with a Zeal swing and count ticks between strikes
- [ ] #2 compare with fid-zeal.md table (7 ticks at 1hs IAS 0)
- [ ] #3 if mismatching, local fork branch with the fix and a zero-mismatch rerun
<!-- AC:END -->

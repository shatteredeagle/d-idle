---
id: TASK-1055
title: 'skillcapretire: retire resource.skillCapPerLevel'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T19:07:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 511000
card: "skillcapretire"
lane: next
owns:
  - "internal/points"
  - "internal/resource"
  - "internal/env/tuning_resource.go"
  - "cmd/d2bot/pointswire.go"
  - "cmd/d2bot/resource_test.go"
  - "data/darklaunch.json"
  - "docs/points.md"
  - "docs/knobs.md"
  - "docs/process/points-verbs-devp14.md"
exclusive: []
contracts: []
result: "resource.skillCapPerLevel (default 0), its level-cap check, CodeLevelCap and tests removed; ledger row retired; knobs.md regenerated. Offline tests only, not verified live; default behaviour unchanged. closecards3: status set to Done, merged on main and every acceptance box ticked."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 knob, level-cap check and tests removed; default behaviour identical
- [x] #2 ledger row retired, darklaunch check passes, knobs.md regenerated
<!-- AC:END -->

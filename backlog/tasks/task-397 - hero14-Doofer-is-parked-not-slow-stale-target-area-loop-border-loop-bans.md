---
id: TASK-397
title: 'hero14: Doofer is parked, not slow (stale target-area loop, border-loop bans)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-14T10:00:00Z'
updated_date: '2026-11-14T10:12:00Z'
labels:
  - bot
  - research
dependencies: []
priority: medium
ordinal: 248000
card: "hero14"
result: "analysis doc only, from the container log, not verified live; root cause: stale ob.targetArea in Dark Wood trips the border-loop breaker every tick, bans Blood Moor and Stony Field for 48 min, quest route refused; 0 kills in the last 14 h"
owns:
  - "docs/probes/hero14-analysis-2026-11-14.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: the fix cards agent/staletarget, agent/questpark, agent/sellskip, agent/goldprio2 (coordinator creates them). Analysis of the sandbox hero container log 2026-11-13/14; ranked fixes. Done when the doc is merged and the fix cards are created by the coordinator.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Analysis doc written with V/U tags
- [x] #2 Fix proposals ranked with owners
<!-- AC:END -->

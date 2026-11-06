---
id: TASK-33
title: 'testerrt: realtime ladder check on d2d for the tester build'
status: Proposed
assignee: []
created_date: '2026-11-06T14:44:00Z'
labels:
  - G-F
dependencies:
  - TASK-31
priority: high
ordinal: 33000
card: "testerrt"
owns:
  - "docs/tester-realtime.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 (d2d route), gap G6. Audit of lockstep assumptions in the ladder/bot/watch path, then a one-hour realtime session on the worker host with a person answering. Depends on testerd2dimage. Estimate 5 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 list of every place that assumes lockstep or virtual time (HangAfter, the runner no-progress clocks, Hub speed, soak stop) with a verdict each and the fixes needed, each tagged V or U
- [ ] #2 one realtime session of at least 1 h in the container or an equivalent d2d run: stalls, deaths, thoughts shown and answered, CPU and memory with the video open; result says verified live or not and writes after start
<!-- AC:END -->

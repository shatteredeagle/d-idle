---
id: TASK-33
title: 'testerrt: realtime ladder check on d2d for the tester build'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-06T14:44:00Z'
labels:
  - container
  - drill
  - tester
milestone: m-1
dependencies:
  - TASK-31
priority: high
ordinal: 33000
card: "testerrt"
owns:
  - "docs/tester-realtime.md"
exclusive: []
contracts: []
result: "Verified live 2026-11-07: lockstep/virtual-time audit in docs/tester-realtime.md (11 places, none needs a code change for realtime); one realtime session of 61 min in the container with the video open: 0 stalls, 0 deaths, 0 restarts, CPU avg 39.6 percent of a core (max 131.9), memory avg 607 MiB (max 641), 33 thoughts shown, 27 answered, 12363 video frames. Bot play was slow (card TASK-138) and a question repeats (TASK-137). Writes after start: 0 (uncheated, not a gate run)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Slice 1 (d2d route), gap G6. Audit of lockstep assumptions in the ladder/bot/watch path, then a one-hour realtime session on the worker host with a person answering. Depends on testerd2dimage. Estimate 5 h.

Done when: list of every place that assumes lockstep or virtual time (HangAfter, the runner no-progress clocks, Hub...; one realtime session of at least 1 h in the container or an equivalent d2d run: stalls, deaths, thoughts... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 list of every place that assumes lockstep or virtual time (HangAfter, the runner no-progress clocks, Hub speed, soak stop) with a verdict each and the fixes needed, each tagged V or U
- [x] #2 one realtime session of at least 1 h in the container or an equivalent d2d run: stalls, deaths, thoughts shown and answered, CPU and memory with the video open; result says verified live or not and writes after start
<!-- AC:END -->

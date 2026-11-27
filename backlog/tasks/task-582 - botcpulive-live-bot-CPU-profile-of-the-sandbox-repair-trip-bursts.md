---
id: TASK-582
title: 'botcpulive: live bot CPU profile of the sandbox (repair-trip bursts)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-17T03:22:00Z'
labels:
  - ops
  - probe
dependencies: []
priority: medium
ordinal: 301000
card: "botcpulive"
owns:
  - "docs/probes/bot-cpu-live-2026-11-16.md"
exclusive: []
contracts: []
result: "Verified live on the sandbox (no bot code change): calm-window profile only (ledger load 30 percent, IsA 14.5 percent of 8 percent of a core); burst NOT profiled (one-shot profiler); quota kept at 3 (27 percent throttled at 2, 2.4 at 3); profiling off. Remaining (live): AC 2 burst-window profile."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Sandbox profiling per user go 2026-11-16. Done when the doc is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Live baseline CPU profile captured and analysed
- [ ] #2 Burst-window profile captured
- [x] #3 Sandbox restored (profiling off, cpus decided)
<!-- AC:END -->

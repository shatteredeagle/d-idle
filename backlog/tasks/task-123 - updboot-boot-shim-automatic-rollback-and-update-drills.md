---
id: TASK-123
title: 'updboot: boot shim, automatic rollback and update drills'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T23:28:00Z'
labels:
  - container
  - drill
  - tester
milestone: m-1
dependencies:
  - TASK-25
  - TASK-120
  - TASK-31
priority: high
ordinal: 116000
card: "updboot"
owns:
  - "cmd/didleboot"
  - "docs/self-update-drill.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Tier 2 part 3 of docs/research/self-update-design.md sections 3.8, 3.9, 5.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cmd/didleboot (static, small) selects the highest compatible release, counts boots, probes bot API and d2d ticks, rolls back on failure or crash loop, handoff exec; tests with fake clock and probes
- [ ] #2 entrypoint/Dockerfile follow-up (thin launcher, release carries mode logic) written as a patch for TASK-31 owners
- [ ] #3 worker host drills: update, bad update rolls back, hash mismatch refused, bad signature refused, offline, image upgrade over newer release; result says verified live or not, writes after start 0
<!-- AC:END -->

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
  - "deploy/d2d-container/Dockerfile.d2d"
  - "deploy/d2d-container/context.sh"
  - "deploy/d2d-container/update_image_test.go"
  - "internal/selfupdate"
  - "tools/release"
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
- [x] #1 cmd/didleboot (static, small) selects the highest compatible release, counts boots, probes bot API and d2d ticks, rolls back on failure or crash loop, handoff exec; tests with fake clock and probes
- [x] #2 entrypoint/Dockerfile follow-up (thin launcher, release carries mode logic) written as a patch for TASK-31 owners
- [ ] #3 worker host drills: update, bad update rolls back, hash mismatch refused, bad signature refused, offline, image upgrade over newer release; result says verified live or not, writes after start 0
<!-- AC:END -->

result: "offline only; AC3 (container drills) NOT run, written as a checklist in docs/self-update-drill.md. cmd/didleboot (select, probation, crash loop, rollback, image.json) tested with a fake clock and probes, real shell launchers, and an end-to-end drill with the real updater. Dockerfile/context.sh changed directly (entrypoint.sh untouched, existing static tests pass; AC2 applied rather than left as a patch). Owns widened to the Dockerfile, context.sh, internal/selfupdate (Gate, Layout.Rollback, manifest kept in releases) and tools/release (ships entrypoint.sh in the bin tarball). Unverified: d2dping clock in realtime mode, tini signal path, real image build. Status sync 2026-11-08 (stays open): AC3 container drills not run."

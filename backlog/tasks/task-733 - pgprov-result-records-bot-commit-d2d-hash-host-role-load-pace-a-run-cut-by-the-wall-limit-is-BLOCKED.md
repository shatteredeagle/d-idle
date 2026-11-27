---
id: TASK-733
title: >-
  pgprov: result records bot commit, d2d hash, host role, load, pace; a run cut
  by the wall limit is BLOCKED
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:44:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 341000
card: "pgprov"
owns:
  - "internal/scenario/result.go"
  - "cmd/d2scenario/run.go"
  - "internal/scenario/prov.go"
  - "cmd/d2scenario/prov.go"
exclusive: []
contracts: []
result: "Result gains prov (bot commit and dirty flag from the binary VCS stamp or D2LOD_BOT_COMMIT, bot and d2d sha256, host ROLE from D2D_HOST_ROLE, load1, pace) and a bot run without the 'virtual run length reached' line is BLOCKED (both town-start and field paths). Verdict change: runs where the bot exits early without an error were GREEN/RED and are now BLOCKED. Offline tests only (-race -count=20 on both packages); not run live."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Design section 2.2 H5 and H6, 7 card 1. Done when a pair records its provenance and a cut run cannot pass. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Result carries bot commit and dirty flag, d2d binary hash, host role, load, pace
- [x] #2 A run cut by the wall limit (no virtual-run-length-reached line) is BLOCKED, never judged
- [x] #3 must-fail test for the cut run
<!-- AC:END -->

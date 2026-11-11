---
id: TASK-293
title: >-
  viewerfps1: viewer game video toward 12.5-25 fps, stage 1 knobs and the
  framefeed ring
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T02:23:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - bot
  - feature
  - tester
dependencies: []
priority: high
ordinal: 207000
card: "viewerfps1"
owns:
  - "cmd/d2dview"
  - "docs/viewer-fps.md"
exclusive:
  - "fork-branch:d2d.lab3-framefeed"
contracts: []
result: stage 1 d2dview knobs and /api/frames counters plus the ring reader with fallbacks (offline tests), fork framefeed verb d2d.lab3-framefeed 80cc7adc pushed, built and unit-tested on the worker host; NOT verified live (no deploy, no live d2d run). Sandbox measured read-only at 4.3 fps. Not build-specific.
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User requirement 2026-11-10: the viewer's live game video at least 50 percent of the game frame rate (12.5 fps), ideally 20-25, bot pace first. Done when stage 1 and the fork prototype are merged-ready and the deploy checklist is in docs/viewer-fps.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 measure the sandbox video path (fps, ms per shot, bytes, CPU) read-only
- [x] #2 stage 1: d2dview knobs (back-off, JPEG quality, half scale) and frame counters on /api/frames and a log line
- [x] #3 fork verb prototype framefeed (shared ring, no PNG) committed and pushed to the private fork remote
- [x] #4 d2dview reads the ring with fallback to screenshot, offline tests
- [x] #5 docs/viewer-fps.md with measurements, design, risks, staged PRs
<!-- AC:END -->

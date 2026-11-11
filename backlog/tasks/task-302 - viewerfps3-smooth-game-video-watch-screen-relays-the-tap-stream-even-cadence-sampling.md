---
id: TASK-302
title: >-
  viewerfps3: smooth game video, watch screen relays the tap stream,
  even-cadence sampling
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T14:57:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - bot
  - bug
  - tester
dependencies: []
priority: high
ordinal: 210000
card: "viewerfps3"
owns:
  - "cmd/d2dview"
  - "internal/watch/tapcapture.go"
  - "internal/watch/tapcapture_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/ladder_test.go"
  - "docs/viewer-fps.md"
exclusive: []
contracts: []
result: watch video relays the tap stream instead of a 4 fps /frame poll; tap frames evenly spaced (every 3rd 60 Hz draw, 20 fps, game gaps 50 ms, arrival p99 53 ms offline on the worker host); sandbox before measured read-only (tap 24.2 fps, gaps p99 62 ms, d2d draws 60 Hz); NOT verified live after (no deploy). Not build-specific.
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User 2026-11-11: stream smoother but not smooth, then sub 1-2 fps. Done when the watch video follows the tap rate with even game-time spacing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 measure pacing on the sandbox read-only: ring publish gaps, stream arrival gaps, the watch path
- [x] #2 watch screen game video relays the tap's /stream.mjpg instead of polling /frame at 4 fps
- [x] #3 tap samples on an even divisor of the source rate, notifies stream clients, stamps game ms
- [x] #4 offline tests; whole-repo gate on the worker host
- [x] #5 docs/viewer-fps.md section 7 with before numbers and the deploy check
<!-- AC:END -->

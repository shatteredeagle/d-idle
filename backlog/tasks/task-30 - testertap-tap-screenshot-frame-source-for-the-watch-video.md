---
id: TASK-30
title: 'testertap: tap screenshot frame source for the watch video'
status: Proposed
assignee: []
created_date: '2026-11-06T14:44:00Z'
labels:
  - G-F
dependencies: []
priority: high
ordinal: 30000
card: "testertap"
owns:
  - "internal/watch/tapcapture.go"
  - "internal/watch/tapcapture_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 of docs/research/tester-build-plan.md (d2d route), gap G4. The d2d container has no X display, so the watch video needs a CaptureFunc fed by the tap's screenshots. Depends on nothing; estimate 5 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 watch.TapCapture returns a CaptureFunc of JPEG frames from the tap (poll of its /frame with the tap started -jpeg, or the read-only socket plus PNG to JPEG); frame rate 1 to 6, stops when the context ends, a tap that is down gives an error and no spin; tests with a fake tap, no real d2d
- [ ] #2 result states the measured bytes per frame and the CPU of one stream (offline numbers; live numbers belong to testerrt)
<!-- AC:END -->

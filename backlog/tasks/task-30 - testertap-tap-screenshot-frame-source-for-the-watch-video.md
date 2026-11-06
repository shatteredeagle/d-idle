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
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/ladder_test.go"
  - "cmd/d2dview/main.go"
  - "deploy/d2d-container/entrypoint.sh"
  - "deploy/d2d-container/Dockerfile.d2d"
  - "docs/d2d-view.md"
  - "docs/d2d-container-bot.md"
  - "docs/d2d-container/README.md"
  - "docs/streamctl-d2d.md"
  - "tools/d2dview-demo.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 of docs/research/tester-build-plan.md (d2d route), gap G4. The d2d container has no X display, so the watch video needs a CaptureFunc fed by the tap's screenshots. Depends on nothing; estimate 5 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 watch.TapCapture returns a CaptureFunc of JPEG frames from the tap (poll of its /frame with the tap started -jpeg, or the read-only socket plus PNG to JPEG); frame rate 1 to 6, stops when the context ends, a tap that is down gives an error and no spin; tests with a fake tap, no real d2d
- [x] #2 result states the measured bytes per frame and the CPU of one stream (offline numbers; live numbers belong to testerrt)
<!-- AC:END -->

## Notes
- Owns widened by the coordinator's brief: the flag `--watch-video-source x11|tap` (+ `--watch-tap-url`, env twins) in cmd/d2bot/ladder.go, and the tap's default moved from :8891 to loopback 127.0.0.1:8893 (d2dview flag/env `D2DVIEW_ADDR`, entrypoint, docs, demo script) so the watch screen owns 8891/8892.
- Does /frame keep the sampler awake? V (code read, cmd/d2dview/main.go `frame`, frames.go `Touch`/`watching`): yes, each /frame request touches the sampler, which runs while the last touch is under 5 s old. Polling at 1 to 6 per second keeps it awake; no viewer, no poll, it sleeps. Live behaviour U.
- Offline cost (TestTapFrameCost, synthetic 800x600 noisy gradient, quality 5 = JPEG 85): PNG 613940 B -> JPEG 98780 B, 36 ms per conversion, about 14 percent of one core at the default 4 fps (the worker host under load; U for a real game screenshot, which compresses differently). Live numbers belong to testerrt.

result: offline tested with httptest fake taps (PNG and JPEG, wrong token, 404, 503 then frame, tap down, tap lost mid-stream, on-demand start and stop, 3-viewer cap, 512 KiB cap, two instances); NOT verified live against a real d2dview or d2d.

---
id: TASK-23
title: 'testerwatch: separate watch token, tap video source switch'
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
updated_date: '2026-11-06T14:55:00Z'
labels:
  - G-F
dependencies:
  - TASK-30
priority: high
ordinal: 23000
card: "testerwatch"
owns:
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/ladder_test.go"
exclusive: []
contracts: []
result: "AC 1 done offline (cmd/d2bot tests: token file 0600 reuse, viewer and answer route reject the bot API token, env-only enablement); NOT verified live; AC 2 (video source) and 3 wait for TASK-30 and the entrypoint/README edits, which this task does not own"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 of docs/research/tester-build-plan.md (d2d route, revised 2026-11-06), gaps G3 and G4. Reworked from the Wine version: the 832x640 slot display criterion moved to the Wine appendix (Wine route, later/optional). Depends on testertap (TASK-30). Estimate 4 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 d2bot reads the watch token from D2BOT_WATCH_TOKEN or a token file (never --token on a command line) and the bot API token stays independent; tests prove the viewer is gated and the API is not
- [ ] #2 d2bot gets --watch-video-source x11|tap (env D2BOT_WATCH_VIDEO_SOURCE, default x11); with tap the video is fed by watch.TapCapture and no ffmpeg or X display is needed; tests for both values and for an unknown value (error naming it)
- [ ] #3 docs/research/tester-build-plan.md gaps G3 and G4 updated with the result
<!-- AC:END -->

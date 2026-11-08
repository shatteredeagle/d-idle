---
id: TASK-23
title: 'testerwatch: separate watch token, tap video source switch'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-06T14:27:00Z'
updated_date: '2026-11-08T03:49:00Z'
labels:
  - watch
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-30
priority: high
ordinal: 23000
card: "testerwatch"
owns:
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/ladder_test.go"
overlap: "TASK-44 (onboardA, 2026-11-06) adds one flag (--watch-access-file) to cmd/d2bot/ladder.go and one test to ladder_test.go; the watch-token tests are unchanged."
exclusive: []
contracts: []
result: "Status 2026-11-07 (decpass2): all boxes ticked after the first end-to-end container run (docs/tester-live.md, TASK-25); AC 3 = plan G3/G4 result text in docs/research/tester-build-plan.md. Earlier: AC 1 done offline (cmd/d2bot tests: token file 0600 reuse, viewer and answer route reject the bot API token, env-only enablement); NOT verified live; AC 2 (video source) and 3 wait for TASK-30 and the entrypoint/README edits, which this task does not own Status sync 2026-11-07: stays open, AC 2 and 3 need the real container and the tester live run (TASK-25 pending)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Slice 1 of docs/research/tester-build-plan.md (d2d route, revised 2026-11-06), gaps G3 and G4. Reworked from the Wine version: the 832x640 slot display criterion moved to the Wine appendix (Wine route, later/optional). Depends on testertap (TASK-30). Estimate 4 h.

Done when: d2bot reads the watch token from D2BOT_WATCH_TOKEN or a token file (never --token on a command line) and...; d2bot gets --watch-video-source x11|tap (env D2BOT_WATCH_VIDEO_SOURCE, default x11); with tap the video is...; docs/research/tester-build-plan.md gaps G3 and G4 updated with the result (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 d2bot reads the watch token from D2BOT_WATCH_TOKEN or a token file (never --token on a command line) and the bot API token stays independent; tests prove the viewer is gated and the API is not
- [x] #2 d2bot gets --watch-video-source x11|tap (env D2BOT_WATCH_VIDEO_SOURCE, default x11); with tap the video is fed by watch.TapCapture and no ffmpeg or X display is needed; tests for both values and for an unknown value (error naming it)
- [x] #3 docs/research/tester-build-plan.md gaps G3 and G4 updated with the result
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Groom 2026-11-06: AC 2 ticked: --watch-video-source x11|tap is on main (cmd/d2bot/ladder.go, tests for both values and an unknown one). Blocking Done: AC 3 (update plan gaps G3 and G4 with the result).
<!-- SECTION:NOTES:END -->

---
id: TASK-336
title: 'videoresume: game video recovers from stalls, hidden tabs and dead viewers'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T20:10:00Z'
labels:
  - watch
  - bug
  - tester
dependencies: []
priority: high
ordinal: 225000
card: "videoresume"
owns:
  - "internal/watch/web/panels/video"
  - "internal/watch/testdata/video_panel.js"
  - "internal/watch/video.go"
  - "internal/watch/web/mjpeg.js"
  - "internal/watch/http_view.go"
  - "internal/watch/testdata/panel_harness.js"
  - "internal/watch/videoresume_test.go"
  - "docs/viewer-fps.md"
exclusive: []
contracts: []
result: page stall watchdog, backoff reconnect, hidden-tab pause and resume, Retry, window.d2video hook (node tests on a fake clock); per-part write deadline frees a dead viewer (Go test fails without it); no server timeout found. NOT verified live (no deploy).
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User 2026-11-11: the video freezes if it sits too long. Done when the page recovers by itself and a dead viewer cannot hold a slot.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 page: stall watchdog with backoff reconnect, reconnecting and stream-down badges, Retry
- [x] #2 page: hidden tab closes the stream after 10 s, visible/focus/pageshow/online reopen it; window.d2video.reconnect()
- [x] #3 server: per-part write deadline frees a dead viewer's slot; no global timeouts
- [x] #4 node and Go tests
<!-- AC:END -->

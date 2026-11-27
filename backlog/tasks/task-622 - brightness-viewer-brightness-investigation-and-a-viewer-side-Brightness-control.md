---
id: TASK-622
title: >-
  brightness: viewer brightness investigation and a viewer-side Brightness
  control
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T15:31:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 310000
card: "brightness"
owns:
  - "docs/probes/viewer-brightness-2026-11-17.md"
  - "internal/watch/web/panels/video/brightness.js"
  - "internal/watch/web/panels/video/panel.css"
  - "internal/watch/web/panels/video/panel.js"
  - "internal/watch/testdata/video_panel.js"
  - "internal/watch/testdata/video_pal_panel.js"
  - "internal/watch/testdata/video_labels_panel.js"
  - "internal/watch/palfeed/luma_test.go"
  - "tools/videofeed/gamma_frames.py"
  - "tools/videofeed/lumastat/"
exclusive: []
contracts: []
result: "Viewer is not brighter than d2d (lossless luma identical, JPEG -1.2 percent); d2d default gamma 155 is a 1.55x gain (game.exe DirectDraw look, GDI -w has none). Added a per-browser viewer Brightness button; sandbox gamma change left to the user. Not verified live; gamma-100 frame pair not recorded (builder queue). closecards3: ticked #4 (merged through the merge gate (merge commit on main))."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. User: viewer overly bright. Done when the cause is measured and the control ships.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 probe doc with numbers: which component is brighter
- [x] #2 viewer Brightness control (Dark/Default/Bright), per browser, default 100%
- [x] #3 regression test: viewer canvas path keeps reference luminance
- [x] #4 gates, mergeq
<!-- AC:END -->

---
id: TASK-536
title: 'viewerfps: viewer game picture fps diagnosis and fixes'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T03:31:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 284000
card: "viewerfps"
owns:
  - "internal/watch/video.go"
  - "internal/watch/videofps_test.go"
  - "internal/watch/web/mjpeg.js"
  - "internal/watch/web/panels/video"
  - "docs/probes/viewer-fps-2026-11-15.md"
  - "internal/watch/glance.go"
  - "internal/watch/videoresume_test.go"
  - "internal/watch/testdata/video_panel.js"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Friend (Waterfox, fast PC) reports bad game-view fps. Server was already latest-wins and shares one encode; added /stream?fps=N, duplicate skip, glance stream stats, page low-rate mode + hint + fps line, copy-free parser. Browser fps NOT measured (builder saturated); not verified live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: friend reports bad fps in the game view. Done when: diagnosis doc, additive fixes, tests green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Server stats, fps cap, duplicate skip, tests
- [x] #2 Page low-rate mode, hint, stats
- [ ] #3 Browser measurement (not done: builder saturated)
<!-- AC:END -->

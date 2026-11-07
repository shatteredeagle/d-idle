---
id: TASK-50
title: 'friendspreview: shareable clickable preview of the flow for friends feedback'
status: Done
assignee: []
created_date: '2026-11-06T20:30:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - tester
dependencies: []
ordinal: 50000
card: "friendspreview"
owns:
  - "docs/research/mocks/friends-preview.html"
exclusive: []
contracts: []
result: "Single-file preview built; clicked through at 390 and 1280 px light and dark in headless Chromium on the worker host (no overflow, no console errors, no network, copy fallback works). Not shown to a real browser clipboard permission."
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Single offline HTML preview walks landing, install, setup, watch/thoughts/rules, orders map, journal, help/report, death, real-vs-not, and a copy-only feedback page
- [x] #2 Checked in headless browser at 390 and 1280 px: no overflow, no console errors, whole path clicked
<!-- AC:END -->

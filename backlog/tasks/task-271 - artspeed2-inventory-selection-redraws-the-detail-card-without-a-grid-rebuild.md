---
id: TASK-271
title: 'artspeed2: inventory selection redraws the detail card without a grid rebuild'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T21:24:00Z'
updated_date: '2026-11-11T21:25:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 198000
card: "artspeed2"
owns:
  - "internal/watch/web/inventory.js"
  - "internal/watch/testdata/inventory_dom.js"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: draw(last) skipped on click because the view JSON is unchanged (selection is not in the signature), so the detail card stayed stale (real, from the TASK-268 skip). Fix: select() moves the .sel class and redraws the detail card only. Result: node DOM test (stub), not run in a browser.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Click selection refreshes the detail card and highlight; grids and focus untouched
- [x] #2 Poll skip on unchanged JSON kept
<!-- AC:END -->

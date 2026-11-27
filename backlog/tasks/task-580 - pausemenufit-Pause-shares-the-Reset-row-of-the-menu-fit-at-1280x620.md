---
id: TASK-580
title: 'pausemenufit: Pause shares the Reset row of the menu (fit at 1280x620)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T03:05:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 300000
card: "pausemenufit"
owns:
  - "internal/watch/web/pauseui.js"
  - "internal/watch/web/shell.css"
exclusive: []
contracts: []
result: "Pause moved into the Reset row of the menu; default-pause* fit pages green on Tower (all sizes incl. 1280x620). Full fit result in the merge report."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

The Pause item at the top of the menu added a row and covered the last item at 1280x620; it now shares the Reset row.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 default-pause-menu fits at 1280x620 and every size
- [ ] #2 full fit green
<!-- AC:END -->

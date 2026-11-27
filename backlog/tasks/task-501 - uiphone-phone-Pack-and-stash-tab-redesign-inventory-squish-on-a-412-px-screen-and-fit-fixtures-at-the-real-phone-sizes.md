---
id: TASK-501
title: >-
  uiphone: phone Pack and stash tab redesign (inventory squish on a 412 px
  screen) and fit fixtures at the real phone sizes
status: Done
assignee:
  - credit-runner
created_date: '2026-11-15T15:00:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 276000
card: "uiphone"
owns:
  - "internal/watch/web/inventory-canvas.css"
  - "internal/watch/web/inventory-canvas.js"
  - "internal/watch/web/inventory.js"
  - "internal/watch/invui.go"
  - "internal/watch/testdata/charscreen/"
  - "tools/uishots/"
  - "tools/uishots.sh"
exclusive: []
contracts: []
result: "Phone Pack and stash tab redesigned for the 2026-11-15 screenshot (412 px, Chrome): both phone tabs scroll under a fixed header, bar and tab row; pack, belt and stash take the full width (cell = width / columns, never under 32 px); art cells show only the picture (the cell font no longer wins over font-size 0, a shared fix); selected item is a bottom sheet with 44 px buttons; one chip row beside Repair and Sell; errands fold into one row; the lock line wraps; a pages menu keeps Inventory, Skills, Policy and Return; a live belt whose items carry slot numbers as x is laid out in four columns (belt slot-number layout is U: seen only in the screenshot). Hero tab also scrolls and uses the width cell (shared fix, 360 px tiles three across). Fit fixtures at 412x780, 393x700, 360x640, 412x890, 390x844 with fixture art and real-looking data; six must-fail pages (label over art, clipped button row, horizontal overflow, cut chips, truncated lock line, small cells) fail as they must; full tools/uishots.sh --fit green (874 ok). Renders looked at 412x780, 360x640, 390x844 dark and light. Not verified on a real phone or with real game art. closecards3: ticked #4 (merged through the merge gate (merge commit on main))."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Phone storage tab: grids use the full width (cell at least 32 px), art cells show only the picture, belt row and stash full width, selected item as a bottom sheet with 44 px buttons, one chip row with Repair and Sell, folded town errands, the lock line wraps, header keeps a pages menu
- [x] #2 Fit fixtures at 412x780, 393x700, 360x640, 412x890 and 390x844 with real-looking data (art, full pack with 2x3 and 2x4 items, seven-potion belt, long name, sheet, errands); must-fail pages for label over art, clipped button row, horizontal overflow, truncated lock line, cut chips, small cells
- [x] #3 tools/uishots.sh --fit passes at every size and the renders at 412x780, 360x640 and 390x844 (dark and light) were looked at
- [x] #4 the whole-repo gate passes on the builder
<!-- AC:END -->

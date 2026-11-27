---
id: TASK-617
title: 'morefit: the Orders More list never clips on a short window'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T14:43:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 310000
card: "morefit"
owns:
  - "internal/watch/web/panels/orders/panel.css"
  - "tools/uishots/fixtures_menu.go"
exclusive: []
contracts: []
result: "The More list is no longer flex-shrunk to zero (min five rows, the sheet scrolls); fit measures the visible rows and clipping; pages more-* at five short sizes plus a must-fail page. Fit and pictures: see the merge report. Closed by closecards2: merged, ACs verified against the merge diff and tests."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User screenshot 2026-11-17: More showed one row at 870x520 with a Needs you row above.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 the More list keeps at least five rows; a short window scrolls the sheet
- [x] #2 fit check: min(3, commands) rows visible, no clip without scroll; must-fail moredefect-clip
- [x] #3 pages at 870x520, 1000x560, 1100x600, 1280x480, 1280x620 with Needs you and with 3+ commands
<!-- AC:END -->

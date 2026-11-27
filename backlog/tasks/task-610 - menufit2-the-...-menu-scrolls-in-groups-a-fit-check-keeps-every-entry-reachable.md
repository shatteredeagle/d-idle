---
id: TASK-610
title: 'menufit2: the ''...'' menu scrolls in groups; a fit check keeps every entry reachable'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T10:52:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 307000
card: "menufit2"
owns:
  - "internal/watch/web/shell.css"
  - "tools/uishots/fixtures_menu.go"
  - "docs/ui-overlays.md"
exclusive: []
contracts: []
result: "Menu = primary row + one scrolling body of collapsible groups (Hero, Tools, Layout); fit.js measures reachability by scrolling, the primary row, scroll possible and hinted; three must-fail defects (noscroll, primary, hint) and healthy tall-menu pages; Orders More list must scroll; vault-reveal-multi pages for the cost line. fit run status in the merge report. Closed by closecards2: merged, ACs verified against the merge diff and tests."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

The menu failed the 1280x620 fit repeatedly as entries were added. Grouped scrolling body plus a reachability check.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 primary row (Pause, Reset) always on show; groups scroll inside the window with a hint
- [x] #2 fit check: every menu entry reachable, scroll possible and hinted; must-fail defect pages
- [x] #3 vault multi-select cost line fixture
<!-- AC:END -->

---
id: TASK-454
title: 'uis6: The area map, the drawers, and the phone arrangement'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T22:37:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 260000
card: "uis6"
owns:
  - "internal/watch/web/panels/area-map/"
  - "internal/watch/web/panels/log/"
  - "internal/watch/web/panels/map/"
  - "internal/watch/testdata/areamap/"
  - "internal/watch/testdata/logpanel/"
  - "internal/watch/logtab.go"
  - "internal/watch/logtab_test.go"
  - "internal/watch/areamap_test.go"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/components.css"
  - "internal/watch/web/layout.js"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/layouts/cockpit.json"
  - "internal/watch/layout.go"
  - "internal/watch/panels.go"
  - "internal/watch/serve.go"
  - "tools/uishots/"
  - "tools/uishots.sh"
  - "tools/uilint/lint.go"
exclusive: []
result: "Built, NOT verified live. Fit check (labrun tower, tools/uishots.sh --fit): 176 of 176 ok = 22 pages x 8 sizes (1920x1080 1440x900 1366x768 1914x930 1904x1000 1536x730 1280x620 390x844) incl. the three phone tabs, the log and first-steps drawers open, the map switch and Layers popover hittable, and the must-fail fixtures failing as they must. Per-package go test of internal/watch, tools/uilint, tools/uishots green; uilint clean. Renders looked at (uishots --cockpit): demo and real cockpit at 1914x930, 1920x1080, 1440x900, 1280x620; phone Orders and Map tabs; log and first-steps drawers at 1440 and 390. Critique: (1) the area map is whole and letterboxed, but in the tall narrow rail tile a wide room leaves large empty bands above and below; (2) at 1280x620 the stand-in act map rows are about 20 px with 4 needs rows, S4/S5 need a slotMin; (3) phone band wraps Return screen onto two lines (S2). Build-specific: none. Old layouts unchanged: area-map and log are cockpitOnly (selectPanels skips them elsewhere); map panel untouched, so its goldens and Classic are unchanged (cells golden and Compat tests green; pixel diff not run). Not done: map drawing module shared with map (area-map has its own copy of the draw functions), Needs you info row for Getting started (needs S3: use ctx.openDrawer('onboarding')), sentence popover link to the log (S2: ctx.openDrawer('log')). Remaining: AC 2 map panel goldens and Classic 1440 pixel diff. 2026-11-27: AC 2 stays open: the map panel golden tests are green in go test ./internal/watch, but Classic was deleted by E11a so the pixel diff against it cannot run; what is left is a uishots diff of the cockpit at 1440 against the worker host golden (browser, not run here)."
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 DOM goldens of area-map (with trail, empty, no data) and of the drawer per tab
- [ ] #2 The map panel's goldens are unchanged and Classic at 1440 is pixel-identical (uishots diff)
- [x] #3 At 1920x1080 the rail shows both maps; at 1440x900 the switch shows one at a time; the fit check passes in both
- [x] #4 At 390x844 each tab fits the screen under the picture with the fixture data (render looked at)
<!-- AC:END -->

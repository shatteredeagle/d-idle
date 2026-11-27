---
id: TASK-424
title: 'uis1: Shell foundations and the default flip'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T15:47:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 255000
card: "uis1"
owns:
  - "internal/watch/web/shell.js"
  - "internal/watch/web/layout.js"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/layout.json"
  - "internal/watch/web/layouts/"
  - "internal/watch/web/tokens.css"
  - "internal/watch/web/shell.css"
  - "internal/watch/layout.go"
  - "internal/watch/layout_test.go"
  - "tools/uilint/"
  - "tools/uishots/layout-rows.json"
  - "tools/uishots/genlayouts.js"
exclusive: []
contracts: []
result: "Built offline, then checked on the build host (headless Chromium, fixture viewer, no live game): whole-repo vet + short tests + race on internal/watch green after merging origin/main. Browser: ?layout=classic keeps every cell of the page before S1 (pictures at 390 and 1920 within 0.5 percent of the pre-S1 golden; at 768, 1280 and 1440 the pixel diff is larger only because the band is now one row high with the ... menu and everything rises about 14 px, same cells); overlay-1920 identical to golden (R2). Standard and Simple looked at at 1440 and 390: the fallbacks (Gear, Hero now, act map, Why, Tally, Thoughts) fill the new tiles' cells. Not done: scripted click-through of the ... menu, Save and Reset in a browser (U), the Electron viewer (needs a display). Review R1 to R5 applied. The offer of Standard after onboarding is a later slice. Files outside owns: panels.go, http_view.go, testdata/layout_s1.js, testdata/layout_cells_s1.json. Remaining: ACs 5 and 6 (preset switch without POST, uilint and pictures). 2026-11-27 follow-up: AC 5 guard test TestChoosePresetMakesNoRequest (the Layout menu stores the choice and reloads, no request; Edit layout save and Reset are the older paths, not re-run live). AC 6 left open: uilint (status, fallback, optIn: tools/uilint tests) and the token compat test are green, but Standard, Simple and Classic were deleted by E11a, so the 390 and 1440 pictures of them cannot be taken; only the cockpit and stream pictures remain, a browser check on the worker host."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A test loads `testdata/compat/layout-v1-{mockup,live,command}.json` and the old `layout.json` (now classic) and asserts the same placed ids and cells as before this change
- [x] #2 Standard and Simple resolved against today's panel list: no unknown id, no overlap, no empty cell; every fallback of D.4 is shown in its cell (node test of the plan)
- [x] #3 An `optIn` panel is not mounted under a layout that does not name it (node test with a fixture manifest)
- [x] #4 `?layout=classic` renders the page as before S1 (uishots diff against the present golden, said in the result)
- [x] #5 The `...` menu switches presets without a POST; Edit layout still saves and Reset returns to the chosen preset
- [ ] #6 uilint knows `status`, `fallback`, `optIn`; token compat test green; pictures at 390 and 1440 looked at for Standard, Simple and Classic
<!-- AC:END -->

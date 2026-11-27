---
id: TASK-616
title: 'jewelicons: jewels show a white diamond in the stash'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T14:36:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 310000
card: "jewelicons"
owns:
  - "internal/watch/web/inventory.js"
  - "internal/watch/web/components.css"
  - "internal/status/inventory.go"
  - "cmd/d2bot/display.go"
  - "tools/uishots/fixtures_phone.go"
  - "docs/inventory-ui.md"
  - "internal/watch/web/inventory-canvas.css"
  - "internal/data/invfile_real_test.go"
  - "cmd/d2bot/itemart_test.go"
  - "tools/uishots/fit_char_test.go"
result: "Cause (S, no install on the build hosts): the icon was already picked by base class; the game jewel art is untinted grey-white, so all jewels looked alike. Fix: status.Item.kind, jewel cells tinted by quality, type-letter fallback for items with no icon, console debug log once per code, fixture, TestRealIconAudit (not run: no install). Pictures viewed on the worker host, not live. closecards3 Remaining (live): #4 need the live game, a browser or a worker-host run; not verified here."
exclusive: []
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
- [x] #1 Cause found and stated
- [x] #2 Jewels tinted by quality, missing icons show a type glyph, logged once in debug
- [x] #3 Audit of item codes against the icon set listed in docs
- [ ] #4 Pictures viewed at 1366x768 and 412x780, dark and light
<!-- AC:END -->

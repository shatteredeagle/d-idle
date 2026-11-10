---
id: TASK-269
title: 'itemtip: D2-style item tooltip on the inventory page'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T21:02:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 197000
card: "itemtip"
owns:
  - "internal/watch/web/inventory-tip.js"
  - "internal/watch/web/inventory-tip.css"
  - "internal/watch/itemtip_test.go"
  - "internal/watch/testdata/itemtip"
  - "docs/inventory-tooltip.md"
  - "internal/status/inventory.go"
  - "internal/status/itemtip_test.go"
  - "internal/status/testdata/item_tip.json"
  - "internal/invdesk"
  - "internal/d2dsrc/tooltips.go"
  - "internal/d2dsrc/tooltips_test.go"
  - "internal/d2dsrc/source.go"
  - "internal/d2dsrc/items.go"
  - "internal/d2denv/reader.go"
  - "internal/d2denv/env.go"
  - "cmd/d2bot/itemtipwire.go"
  - "cmd/d2bot/itemtipwire_test.go"
  - "cmd/d2bot/display.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "internal/watch/invapi.go"
  - "internal/watch/invui.go"
  - "internal/watch/web/inventory.js"
  - "internal/watch/web/tokens.css"
exclusive: []
contracts: []
result: "Offline only, not verified live or in a browser. Tooltip component on /watch/inventory (hover, tap, focus, Escape, viewport placement, no scroll calls), API schema 2 with optional tip, base and sell item fields, d2d item text carried from the stock items verb. Magic properties have no structured source (game.exe: base facts and a note only), see docs/inventory-tooltip.md. Whole-repo gate on the worker host exit 0."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Item data audit written, missing data listed
- [x] #2 Hover, tap and focus tooltip on all item cells and doll slots with quality colours
- [x] #3 API gains optional tip fields; golden fixtures; compat unchanged
- [x] #4 Tests: Go, JS pattern, DOM goldens, uilint, whole-repo gate
<!-- AC:END -->

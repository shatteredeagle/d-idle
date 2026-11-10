---
id: TASK-237
title: 'uiinv: inventory page on the shell with the Diablo II paper doll; durability 0 fix; phone checks of policy and return'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-10T01:22:00Z'
labels:
  - watch
  - bug
  - T0
dependencies:
  - TASK-234
priority: high
ordinal: 180000
card: "uiinv"
owns:
  - "internal/watch/invui.go"
  - "internal/watch/invapi.go"
  - "internal/watch/web/inventory.js"
  - "internal/watch/web/inventory.css"
  - "internal/watch/web/components.css"
  - "internal/watch/web/tokens.css"
  - "internal/watch/web/app.css"
  - "internal/watch/http_view.go"
  - "internal/watch/watchlayout_test.go"
  - "internal/watch/testdata/inventory_logic.js"
  - "tools/uishots/"
  - "tools/uishots.sh"
  - "tools/uilint/allow.txt"
  - "docs/ui-design-system.md"
exclusive: []
contracts: []
result: "Inventory page on the shared shell (dark default, top band, theme), Diablo II paper doll, backpack/belt/stash as cell grids with clamped labels, detail card with the old actions, repair card, equip queue; broken items read 0/max BROKEN (node regression test); policy and return checked at 390 px (styled, no overlap; a.btn underline removed). Verified: go test ./internal/watch ./tools/uishots ./tools/uilint, whole-repo vet+test on the worker host, Chrome renders on the worker host at 390 and 1440. Preview deployed; not seen live by the agent (token). Endpoints unchanged."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User screenshot 2026-11-09: the inventory page was white and unstyled (inline style refused by its CSP), durability undefined, overlapping names. Done when the acceptance criteria hold and the preview is up.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 /watch/inventory loads the shared stylesheets and shell (dark default, same top band), no inline style; mobile-first
- [x] #2 a broken item reads 0/max and BROKEN, never undefined (regression test)
- [x] #3 backpack, belt and stash are cell grids whose labels never overlap; the worn slots are the paper doll of the approved mockup
- [x] #4 policy and return render styled and without overlap at 390 px in the screenshot suite
- [x] #5 endpoints and data unchanged; preview deployed
<!-- AC:END -->

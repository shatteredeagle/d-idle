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
  - "internal/itemtip"
  - "internal/watch/web/inventory-tip.js"
  - "internal/watch/web/inventory-tip.css"
  - "internal/watch/itemtip_test.go"
  - "internal/watch/testdata/itemtip"
  - "docs/inventory-tooltip.md"
  - "internal/status/inventory.go"
  - "internal/invdesk"
  - "internal/d2dsrc/tooltips.go"
  - "cmd/d2bot/itemtipwire.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Item data audit written, missing data listed
- [ ] #2 Hover, tap and focus tooltip on all item cells and doll slots with quality colours
- [ ] #3 API gains optional tip fields; golden fixtures; compat unchanged
- [ ] #4 Tests: Go, JS pattern, DOM goldens, uilint, whole-repo gate
<!-- AC:END -->

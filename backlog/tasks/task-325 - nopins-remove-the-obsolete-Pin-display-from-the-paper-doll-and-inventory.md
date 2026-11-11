---
id: TASK-325
title: 'nopins: remove the obsolete Pin display from the paper doll and inventory'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:11:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 221000
card: "nopins"
owns:
  - "internal/watch/web/inventory.js"
  - "internal/watch/web/inventory-actions.js"
  - "internal/watch/web/inventory-tip.js"
  - "internal/watch/nopins_test.go"
  - "docs/inventory-ui.md"
  - "docs/gear-policy.md"
  - "docs/ui-design-system.md"
  - "internal/watch/invui.go"
  - "internal/watch/web/inventory-tip.css"
  - "internal/watch/testdata/inventory_dom.js"
  - "internal/watch/testdata/itemactions/act_dom.js"
  - "internal/watch/testdata/itemtip/tip_dom.js"
exclusive: []
contracts: []
result: "PIN badge, legend chip, Unpin button/menu entry and tooltip badge removed; pins and /api/inventory/unpin kept as deprecated; new .nopin goldens added; not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: under gearDecisions=player the bot never equips, so pins carry no information. Done when: no pin renders; API compat kept.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 No PIN badge, legend chip, Unpin button or menu entry renders
- [ ] #2 pins and unpin endpoint kept in the API as deprecated; old fixtures untouched, new goldens added
- [ ] #3 whole-repo gate green
<!-- AC:END -->

---
id: TASK-382
title: 'mapfog: overlay layers only where the hero has explored'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T20:23:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 242000
card: "mapfog"
owns:
  - "docs/mapfog.md"
  - "internal/senses"
  - "internal/watch/senses.go"
  - "internal/watch/web/panels/map"
  - "internal/watch/testdata/map_panel.js"
  - "tools/uishots"
exclusive: []
contracts: []
result: "Coverage points, target, events and heat in blocks the hero has not explored are withheld server-side (additive hidden key); verified offline and by headless render, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Coverage squares, target, events and heat in unexplored blocks are not sent
- [x] #2 Page shows hidden count; headless render checked
- [x] #3 Tests: field and cave, golden unchanged
<!-- AC:END -->

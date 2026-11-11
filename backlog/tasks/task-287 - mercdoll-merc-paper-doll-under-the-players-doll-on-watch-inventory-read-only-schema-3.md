---
id: TASK-287
title: >-
  mercdoll: merc paper doll under the player's doll on /watch/inventory, read
  only, schema 3
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T01:05:00Z'
updated_date: '2026-11-11T16:24:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 206000
card: "mercdoll"
owns:
  - "internal/watch/web/inventory-merc.js"
  - "internal/watch/web/inventory-merc.css"
  - "internal/watch/testdata"
  - "internal/watch/invui.go"
  - "internal/watch/invapi.go"
  - "internal/watch/mercdoll_test.go"
  - "docs/inventory-ui.md"
  - "docs/live-checks.md"
exclusive: []
contracts: []
result: "Offline only, not seen in a browser (2026-11-10): schema 3 merc block is TASK-286; this adds inventory-merc.js/.css (card under the hero doll, states none, ready, dead, unknown, absent hides it), page and route wiring, golden views and DOM goldens (node), TestMercFixturesDecodeIntoView, TestMercDollRoutesAndRules. No merc actions. Live-checks row: docs/live-checks.md TASK-287 #3."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Stage A of docs/research/merc-gear.md (TASK-283), the UI. No merc actions (stage B/C).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 watch.inventory schema 3 with additive optional merc block
- [x] #2 d-doll under the hero doll with states none, dead, ready, unknown; no actions
- [x] #3 Golden fixtures and DOM tests
<!-- AC:END -->

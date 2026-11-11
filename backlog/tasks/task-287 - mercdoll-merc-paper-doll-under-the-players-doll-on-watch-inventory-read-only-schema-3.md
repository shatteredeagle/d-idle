---
id: TASK-287
title: 'mercdoll: merc paper doll under the player''s doll on /watch/inventory, read only, schema 3'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T01:05:00Z'
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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Stage A of docs/research/merc-gear.md (TASK-283), the UI. No merc actions (stage B/C).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 watch.inventory schema 3 with additive optional merc block
- [ ] #2 d-doll under the hero doll with states none, dead, ready, unknown; no actions
- [ ] #3 Golden fixtures and DOM tests
<!-- AC:END -->

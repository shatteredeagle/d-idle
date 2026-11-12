---
id: TASK-353
title: >-
  mapnodes: Underground Pass, Monastery Gate and Outer Cloister orderable on the
  watch map
status: Done
assignee:
  - credit-runner
created_date: '2026-11-12T03:04:00Z'
updated_date: '2026-11-12T03:23:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 231000
card: "mapnodes"
owns:
  - "internal/watch/mapdata.go"
  - "internal/watch/mapnodes_test.go"
  - "internal/watch/ordersmap_test.go"
  - "internal/watch/testdata"
exclusive: []
contracts: []
result: "Underground Pass (Underground Passage 1, bands 10-16), Monastery Gate and Outer Cloister now carry an Order; test resolves every Order through bot Plan.Resolve (V offline: plan table). Tower Cellars and Monastery line left unorderable: Cellar 1 is not a useful target (Countess is on level 5, whose name has no resolvable id in goals.Catalog) and the line is the main path. Pathing is the nav act1Hints table, not driven live (U). Goldens updated additively (orderable/order/range/why). Web app.js reads n.orderable only, no special case. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Underground Pass node had no Order so the watch map could not send the hero there. Done when: it is clickable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Underground Pass, Monastery Gate, Outer Cloister carry an order name the bot resolves
- [x] #2 test: every node Order resolves through Plan.Resolve
<!-- AC:END -->

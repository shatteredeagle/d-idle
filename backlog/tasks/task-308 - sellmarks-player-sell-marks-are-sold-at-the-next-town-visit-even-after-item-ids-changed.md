---
id: TASK-308
title: 'sellmarks: player sell marks are sold at the next town visit even after item ids changed'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T16:07:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 211000
card: "sellmarks"
owns:
  - "internal/invdesk"
  - "internal/town"
  - "internal/bot/leveling.go"
  - "internal/watch/web/inventory.js"
exclusive: []
contracts: []
result: "offline only (Reconcile, town sell on Paladin and Sorceress); not verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: live 2026-11-11, marked items stayed in the pack: the marks were keyed by item id, which a game restart renumbers, and the sell chore only ran on trips that asked for a sale. Done when: marks are re-attached by base and grid cell, the chore runs whenever a mark is in the pack, the visit logs 'sell: N marked, M sold, K not sold'.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Marks follow items after a game restart
- [x] #2 Sell chore runs on any trip when a mark is in the pack
- [x] #3 Per-visit sell log line and pending count on the Inventory page
<!-- AC:END -->

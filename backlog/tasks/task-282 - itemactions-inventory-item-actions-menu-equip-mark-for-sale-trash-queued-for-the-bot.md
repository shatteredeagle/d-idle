---
id: TASK-282
title: 'itemactions: inventory item actions menu (equip, mark for sale, trash) queued for the bot'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T00:53:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 204000
card: "itemactions"
owns:
  - "internal/invdesk"
  - "internal/watch/web/inventory-actions.js"
  - "internal/watch/web/inventory-actions.css"
  - "docs/inventory-actions.md"
  - "internal/leveler/playertrash.go"
  - "internal/leveler/playertrash_test.go"
  - "internal/loot/ignore.go"
  - "internal/loot/ignore_test.go"
  - "internal/watch/invapi.go"
  - "internal/watch/invapi_test.go"
  - "internal/bot/playeraction.go"
  - "internal/bot/playeraction_test.go"
  - "cmd/d2bot/invwire.go"
exclusive: []
contracts: []
result: "Built offline, NOT verified live: see docs/inventory-actions.md (leveler PlayerTrash and looter ignore tests skip without game tables; live checks listed there)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the player wants to act on items from the watch page: equip, mark for sale, trash. Done when: the menu queues actions the bot performs at an open opportunity, with states, cancel, persistence and tests; nothing verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Equip, Mark for sale and Trash from a per-item action menu, queued with states and cancel
- [x] #2 Trash primitive offline-tested (Paladin and Sorceress), loot ignores trashed items
- [x] #3 Persisted bounded queue, versioned API, goldens, uilint clean
<!-- AC:END -->

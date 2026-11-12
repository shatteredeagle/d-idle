---
id: TASK-380
title: 'gearsets: drag gear onto the paper doll and player-defined gear sets'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T20:13:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 242000
card: "gearsets"
owns:
  - "internal/invdesk"
  - "internal/leveler/playerunequip.go"
  - "internal/leveler/playerunequip_test.go"
  - "cmd/d2bot/invwire.go"
  - "internal/watch/invapi.go"
  - "internal/watch/invapi_test.go"
  - "internal/watch/invui.go"
  - "internal/watch/web/inventory-drag.js"
  - "internal/watch/web/inventory-sets.js"
  - "internal/watch/web/inventory-sets.css"
  - "internal/watch/web/inventory.js"
  - "internal/watch/testdata/gearsets"
  - "internal/watch/gearsets_test.go"
  - "data/unlocks/ladder.json"
  - "docs/gearsets.md"
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
- [ ] #1 Drag a pack tile onto a doll slot queues the same Equip as the button; green/red by the server's per-slot validity
- [ ] #2 Drag a worn item to the pack queues Unequip; merc doll accepts drops and says why it cannot act yet
- [ ] #3 Gear sets: save, rename, delete, preview, apply in a safe order, cancel; never substitutes, never bot-initiated
- [ ] #4 Unlock ladder row gearsets plus docs/gearsets.md; goldens, tests for two classes and the merc
<!-- AC:END -->

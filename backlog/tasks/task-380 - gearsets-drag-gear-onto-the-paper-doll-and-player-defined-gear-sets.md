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
  - "internal/leveler/playerequip_test.go"
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
  - "internal/watch/web/inventory-actions.js"
  - "internal/watch/testdata/itemactions"
  - "data/unlocks/ladder.json"
  - "internal/env/tuning_invdesk.go"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/d2dwire.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/gearsets_test.go"
  - "tools/uishots"
  - "tools/uishots.sh"
  - "docs/knobs.md"
  - "docs/gear-policy.md"
  - "docs/inventory-actions.md"
  - "docs/gearsets.md"
exclusive: []
contracts: []
result: "Built offline, NOT verified live: drag and drop of a backpack tile onto the hero doll (the server validity rule, green or red with the reason), take off by dragging a worn item to the pack, the mercenary doll as a drop target that says why the bot cannot act on him yet (no merc executor exists), the player's gear sets (save, rename, delete, preview, ordered apply as queued steps, cancel; never substituted, never bot-initiated), ladder row gearsets at level 15 and the knob invdesk.maxSets. Node and Go tests for two classes and the merc, goldens, headless pictures looked at."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user wants to drag gear onto the paper doll (hero and mercenary) and keep named gear sets the player defines. The bot never makes an equipment decision (docs/gear-policy.md), so a set is applied only by the player's click. Done when: the criteria below hold; the design, schema and live checks are in docs/gearsets.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Drag a pack tile onto a doll slot queues the same Equip as the button; green/red by the server's per-slot validity
- [x] #2 Drag a worn item to the pack queues Unequip; merc doll accepts drops and says why it cannot act yet
- [x] #3 Gear sets: save, rename, delete, preview, apply in a safe order, cancel; never substitutes, never bot-initiated
- [x] #4 Unlock ladder row gearsets plus docs/gearsets.md; goldens, tests for two classes and the merc
<!-- AC:END -->

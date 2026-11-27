---
id: TASK-930
title: 'equipv2: instant vault equip through direct item verbs'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-25T05:03:00Z'
updated_date: '2026-11-25T05:27:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 432000
card: "equipv2"
owns:
  - "cmd/d2bot/vaultequip.go"
  - "cmd/d2bot/vaultkeep.go"
  - "cmd/d2scenario/run.go"
  - "docs/adr/0029-vault-gold-payment.md"
  - "docs/decisions.md"
  - "docs/vault.md"
  - "internal/d2dctl/"
  - "internal/env/tuning_loot.go"
  - "internal/invdesk/pinslot.go"
  - "internal/vaultkeep/"
  - "internal/watch/"
  - "tools/provingground/scenarios/vaultequipv2.json"
exclusive: []
contracts: []
allow-adr: true
result: "Built offline and Proving Ground GREEN on the devp29 binary (swap and 10-piece wearset, Paladin 11, builder lab). NOT verified live on the sandbox image (needs devp29); Sorceress 3 not run. closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. User request 2026-11-25 (instant vault equip). Done when the one-tap flow wears at once and the sandbox runs the devp29 image.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Equip and Wear set write the items into the body slots through the direct verbs, old pieces become vault records
- [x] #2 Older image or knob 0 falls back to the order flow; measured run refused
- [x] #3 Unit and node tests for two classes pass
- [x] #4 Proving Ground vault.equip-v2-swap and set10 GREEN on devp29
- [ ] #5 Live check on the sandbox image (needs the devp29 image)
<!-- AC:END -->

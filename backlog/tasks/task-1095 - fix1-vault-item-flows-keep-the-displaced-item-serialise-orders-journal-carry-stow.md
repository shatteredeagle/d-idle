---
id: TASK-1095
title: 'fix1: vault item flows keep the displaced item, serialise orders, journal carry/stow'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T03:41:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 524000
card: "fix1"
lane: next
owns:
  - "internal/vaultkeep/"
  - "internal/d2dctl/"
  - "internal/watch/"
  - "docs/design/fork-order-id-spec.md"
exclusive: []
contracts: []
result: "NOT verified live (offline fakes only). Direct Equip and Wear set journal the engine reply before the ledger line and file the displaced item on every failure after the swap; an unanswered call is judged by a pre-read of the slot or stays open (order_in_flight); the vault POST routes answer 409 order_in_flight per record; Carry and Stow have an intent journal and find the charm by its bytes; a credited sale completes with the sale off; twin keeps and twin displaced items are visible. Nothing build-specific (no class or skill data). Left: order id on the wire (docs/design/fork-order-id-spec.md), store-level sale reservation against Keep, Pin and Discard, a merc crash journal."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the one-call vault flows move the player's real items and could lose or double them (docs/design/fable-review-2026-11-28/01-item-flows.md, F1 to F6, F9, F12). Done when: the acceptance criteria hold in the fake-game tests (internal/vaultkeep/fix1_test.go).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 direct Equip/Wear set journal the engine reply before any later step and re-file the displaced item on every failure after the swap
- [x] #2 two vault POSTs for one id serialise, the second answers 409 order_in_flight; EquipMerc create carries an idempotency key
- [x] #3 Carry/Stow write an intent record first, complete or roll back at next start, Stow records before removing and re-resolves the unit
- [x] #4 instant sale reserves records before the credit; a credited order completes with SaleOn false
- [x] #5 presentIn counts re-filed twins; a duplicate Add is visible and not counted as filed
- [ ] #6 live check on a sandbox: Equip from the Vault with a worn item, displaced item is a vault record
<!-- AC:END -->

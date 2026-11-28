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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 direct Equip/Wear set journal the engine reply before any later step and re-file the displaced item on every failure after the swap
- [ ] #2 two vault POSTs for one id serialise, the second answers 409 order_in_flight; EquipMerc create carries an idempotency key
- [ ] #3 Carry/Stow write an intent record first, complete or roll back at next start, Stow records before removing and re-resolves the unit
- [ ] #4 instant sale reserves records before the credit; a credited order completes with SaleOn false
- [ ] #5 presentIn counts re-filed twins; a duplicate Add is visible and not counted as filed
- [ ] #6 live check on a sandbox: Equip from the Vault with a worn item, displaced item is a vault record
<!-- AC:END -->

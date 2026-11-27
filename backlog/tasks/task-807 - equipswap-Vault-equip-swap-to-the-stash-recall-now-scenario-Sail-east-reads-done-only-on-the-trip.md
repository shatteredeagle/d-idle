---
id: TASK-807
title: >-
  equipswap: Vault equip swap to the stash, recall-now scenario, Sail east reads
  done only on the trip
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T15:48:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 369000
card: "equipswap"
owns:
  - "internal/vaultkeep/equip.go"
  - "internal/vaultkeep/equip_test.go"
  - "internal/invdesk/stash.go"
  - "internal/invdesk/stash_test.go"
  - "cmd/d2bot/vaultequip.go"
  - "internal/bot/questboard.go"
  - "internal/bot/questboard_s7b_test.go"
  - "internal/task/act1.go"
  - "internal/task/act2.go"
  - "internal/task/act3.go"
  - "internal/task/act4.go"
  - "tools/provingground/scenarios/manage.json"
  - "docs/vault.md"
  - "docs/questboard.md"
exclusive: []
result: "Sail east: root cause fixed (hop step read Done from Need = not needed while Andariel lives; Evidence is now the world act; must-fail tests on four hops). Equip swap: stash fallback and a room refusal (must-fail tests, Paladin and Sorceress); vault.equip-swap GREEN live on devp19, both classes. recall-now: Paladin GREEN live after the assert fix (it read an unlogged label). recall-now Sorceress 3 stays RED live, twice, to the second: the cap is worn after the equip and found in the backpack 13 s later with no unequip logged between (loot minute); cause open (U), not the equip logic (vault.equip-swap and equip-one-tap are GREEN) and not the chicken reload. closecards3 Remaining (live): #2, #4 need the live game, a browser or a worker-host run; not verified here."
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fixes found by the Vault Equip sandbox test of 2026-11-22. Done when the four boxes are ticked.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Equip swap: displaced item goes to the Vault, else the stash, else stays in the backpack under the keep mark; an order with no home is refused with a reason (must-fail tests, Paladin and Sorceress)
- [ ] #2 manage.recall-now reads the trip's log line (live: Paladin 11 GREEN, Sorceress 3 RED: the worn cap leaves the body, cause open)
- [x] #3 Sail east to Act 2 reads done only when the world is in Act 2 (must-fail tests, four act hops)
- [ ] #4 Scenarios vault.equip-swap and manage.recall-now green on the lab d2d, both classes (live, CHEATED setups)
<!-- AC:END -->

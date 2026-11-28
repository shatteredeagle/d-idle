---
id: TASK-1113
title: >-
  vault2: store reservation for the instant sale, merc crash journal, fix1 path
  tests, ops vault check
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T05:42:00Z'
updated_date: '2026-11-28T07:39:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 533000
card: "vault2"
lane: next
owns:
  - "internal/vault/"
  - "internal/vaultkeep/"
  - "internal/watch/"
  - "cmd/d2bot/opscmd.go"
  - "cmd/d2bot/opsvault_test.go"
  - "docs/design/fork-order-id-spec.md"
  - "docs/vault.md"
exclusive: []
contracts: []
result: "NOT verified live (offline fakes only). The store has an all-or-none, Version-checked Reserve; a held record leaves only by its own order or Release, so the instant sale and Sell all cannot pay for a record that a Keep, Pin, Discard or Reveal then takes. Equip on merc has an intent journal and records the displaced piece before removing it; a two-hander's shield is filed from an other-hand pre-read after a lost reply. Tests inject a stop or lost reply after each step on two classes (sale, twin Keep, merc, Wear set, HTTP 409). d2bot ops vault check is the read-only dry run of a repair. Nothing build-specific (hand slots and merc wear rules come from the Hand). Left: the fork order id (docs/design/fork-order-id-spec.md), whether item full lists merc items (U)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: card fix1 (TASK-1095) left the store-level sale reservation, the merc crash journal and several untested paths open (docs/design/fable-review-2026-11-28/01-item-flows.md F2, F5). Done when: the acceptance criteria hold in the fake-game tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Store.Reserve is an all-or-none, Version-checked compare-and-set; a held record refuses Keep, Pin, Discard, queue Sell and Reveal and leaves only by its own order or Release
- [x] #2 SellNow and Sell all reserve before the credit; a refused credit releases, an unanswered one keeps the hold across a restart, an orphan hold is released
- [x] #3 Equip on merc writes an intent before the first game step, records the displaced piece before removing it and is repaired at the next Run by looking
- [x] #4 a two-hander's shield is filed after a lost reply from the other-hand pre-read; the fork requirement is in docs/design/fork-order-id-spec.md
- [x] #5 tests with injected failures after each step on two classes: instant sale, twin Keep, merc, Wear set, HTTP 409
- [x] #6 d2bot ops vault check reports orphan intents, duplicates and credited-but-unreserved orders read-only
- [ ] #7 live check on a sandbox: sell while tapping Keep, equip on merc with a worn piece, run ops vault check
<!-- AC:END -->

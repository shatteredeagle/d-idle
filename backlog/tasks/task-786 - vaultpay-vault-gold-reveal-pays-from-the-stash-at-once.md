---
id: TASK-786
title: 'vaultpay: vault gold reveal pays from the stash at once'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T19:12:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - feature
  - T2
dependencies: []
priority: medium
ordinal: 361000
card: "vaultpay"
owns:
  - "internal/vaultkeep"
  - "internal/bulkid"
  - "internal/d2dctl/paygold.go"
  - "internal/d2dctl/paygold_test.go"
  - "internal/d2dctl/vault_verbs.go"
  - "internal/d2dctl/vault_fake.go"
  - "internal/watch/vaultapi_gold.go"
  - "internal/watch/vaultapi_gold_test.go"
  - "internal/watch/web/vault.js"
  - "internal/watch/web/vault-logic.js"
  - "internal/watch/web/vault.json"
  - "internal/watch/testdata/vault_reveal.mjs"
  - "internal/watch/testdata/vault_logic.mjs"
  - "cmd/d2bot/vaultkeep.go"
  - "internal/town/playerid.go"
  - "internal/env/tuning_loot.go"
  - "docs/knobs.md"
  - "docs/vault.md"
  - "docs/cheats.md"
  - "docs/adr/0014-memory-writes-allowed-no-crutch.md"
  - "docs/adr/0029-vault-gold-payment.md"
  - "docs/adr/README.md"
  - "docs/decisions.md"
  - "tools/provingground/scenarios/vaultreveal.json"
exclusive:
  - "fork-branch:didle/devp17-20261121"
contracts: []
result: "Gold reveal now PAYS FROM THE STASH AT ONCE (ADR 0029, user popup 2026-11-21): fork verb debug paygold behind --vault-pay (private fork branch didle/devp17-20261121, tests rc 0 incl. paygold_d2d, image d2dbot:devp17 loaded, sandbox untouched, NOT switched); d2dctl.Pay client with a ledger line before the verb and PayFake; the order unlocks held affixes (Ether path), takes the price (vendor price of a Scroll of Identify per item) in ONE paygold call per order id, reveals in the same request, partial in the player's list order, failed with the reason; the scroll-buying path is removed (bulkid chore kept). Built and tested OFFLINE (Paladin and Sorceress, race x20); NOT run against a live game. Scenarios vault.reveal-gold and vault.reveal-gold-short stay pending: a gold reveal is a write that is refused in measured runs, so they need an unmeasured lane first. closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Replaces the scroll-buying gold reveal of task 778 by an instant stash payment (user popup 2026-11-21, ADR 0029). Done when the Reveal bar pays from the stash gold and reveals in one request.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fork verb debug paygold behind --vault-pay: pushed, tested (paygold_d2d), image d2dbot:devp17 loaded, sandbox unchanged
- [x] #2 d2dctl Pay client, ledger line and PayFake; the gold order pays from the stash, reveals at once, no scroll buying
- [x] #3 Must-fail tests on Paladin and Sorceress: pays, partial, overdraft, idempotent, measured refuses, no scroll step
- [x] #4 ADR 0029, docs/vault.md, decisions, knob text, scenarios updated
- [ ] #5 Live check on a --vault-pay d2d with a real hero (not run)
<!-- AC:END -->

---
id: TASK-778
title: 'vaultreveal: pay gold to reveal vault items (the bot buys the scrolls)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T15:01:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - watch
  - feature
  - T2
dependencies: []
priority: medium
ordinal: 359000
card: "vaultreveal"
owns:
  - "internal/vaultkeep"
  - "internal/bulkid"
  - "internal/watch/vaultapi_gold.go"
  - "internal/watch/web/vault.js"
  - "internal/watch/web/vault-logic.js"
  - "internal/watch/web/vault.css"
  - "internal/watch/web/vault.json"
  - "internal/watch/vaultapi.go"
  - "internal/watch/vaultapi_gold_test.go"
  - "internal/watch/vaultreveal_ui_test.go"
  - "internal/watch/testdata/vault_reveal.mjs"
  - "internal/watch/testdata/vault_logic.mjs"
  - "internal/env/tuning_loot.go"
  - "internal/town/playerid.go"
  - "docs/knobs.md"
  - "cmd/d2bot/vaultkeep.go"
  - "docs/vault.md"
  - "tools/provingground/scenarios/vaultreveal.json"
exclusive: []
contracts: []
result: "Gold reveal built OFFLINE, not run live: a gold reveal is an order the bot fills at the next town visit by BUYING Scrolls of Identify at the vendor (real gold above the repair and potion-kit reserve, capped by the order) and reading them through the ordinary Reveal flow in the same visit; no virtual gold. POST /api/vault/revealgold (answer listener, idempotent, one active order, cancel, confirm over loot.vaultRevealGoldConfirm), one Reveal bar replaces the notice paragraph, knob loot.vaultRevealGold default 0. Tests: vaultkeep (two classes), bulkid, watch, node bar tests. Scenarios vault.reveal-gold and vault.reveal-gold-short are DECLARED but pending (framework lacks vault record seeding and vault assertions): no RED or green verdict."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Phone viewer: Bulk identify using gold. Done when the Vault page pays a reveal in gold that the bot spends on real Scrolls of Identify.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Mechanism decided and written in docs/vault.md
- [x] #2 One Reveal bar with a payment selector replaces the notice paragraph
- [x] #3 vault_reveal gold order: POST, idempotent, one active, cancel, status
- [x] #4 Bot buys scrolls at the town visit within the purse guards and reveals the same visit
- [x] #5 knob loot.vaultRevealGold default 0
- [ ] #6 Scenarios vault.reveal-gold and vault.reveal-gold-short (live: unticked if not run)
- [x] #7 Tests on two classes, race, check.sh, leakcheck
<!-- AC:END -->

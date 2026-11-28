---
id: TASK-597
title: 'vaultv7b: wire the ETHER wallet, Vault page upgrade UI, earn call sites'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T06:53:00Z'
labels:
  - watch
  - feature
  - T0
dependencies: []
priority: medium
ordinal: 304000
card: "vaultv7b"
owns:
  - "internal/ether"
  - "internal/watch/etherapi.go"
  - "cmd/d2bot/etherwire.go"
  - "docs/ether.md"
  - "data/vault/ether.json"
  - "internal/watch/web/vault.js"
  - "internal/watch/web/vault.json"
  - "internal/watch/web/vault.css"
  - "tools/uishots/fixtures_vault.go"
  - "tools/uishots/fit.js"
  - "tools/uishots/fit.go"
  - "tools/uishots/fit_char_test.go"
  - "tools/uishots/serve.go"
  - "tools/uishots.sh"
  - "cmd/d2bot/vaultpage.go"
  - "cmd/d2bot/etherwire_test.go"
  - "internal/watch/etherapi_test.go"
  - "internal/watch/vaultether_test.go"
  - "internal/watch/testdata/vault_ether.mjs"
  - "internal/watch/vaultapi.go"
  - "internal/watch/vaultui_test.go"
  - "internal/watch/hero.go"
  - "internal/watch/web/panels/hero-vitals/panel.js"
  - "docs/vault.md"
  - "docs/design/loot-vault-2026-11-16.md"
exclusive: []
contracts: []
result: "Built and tested offline, not run live: the ETHER wallet is wired in cmd/d2bot behind loot.vault (per account, survives hero switches, feeds store capacity), gated POST routes, Vault page Upgrade/Refund with wallet chip and confirm sheet (node test, fit pages and pictures viewed on the builder), Hero tile chip, earn call sites (firsts, bosses, level and difficulty milestones, capped kill trickle). Quest hand-in earn stays S10."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the V7 wallet was built but not wired; the player cannot earn or spend Ether. Done when: with loot.vault on the account wallet earns from firsts, bosses, milestones and a capped kill trickle, buys and refunds capacity from the Vault page, and nothing changes with the knob at 0 (docs/ether.md).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Wallet wired in cmd/d2bot behind loot.vault, per account, survives hero switches
- [x] #2 Vault page Upgrade/Downgrade/chip, fit and pictures
- [x] #3 Earn call sites idempotent, tests
<!-- AC:END -->

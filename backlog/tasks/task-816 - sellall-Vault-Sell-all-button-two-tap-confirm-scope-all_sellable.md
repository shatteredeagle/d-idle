---
id: TASK-816
title: 'sellall: Vault Sell all button, two-tap confirm, scope all_sellable'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T19:39:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 372000
card: "sellall"
owns:
  - "internal/vaultkeep"
  - "internal/watch"
  - "cmd/d2bot/vaultkeep.go"
  - "docs/vault.md"
  - "docs/ui/vault-sell-all"
  - "tools/provingground/scenarios/vaultsellall.json"
  - "tools/uishots"
  - "tools/uishots.sh"
exclusive: []
contracts: []
lane: next
result: " Remaining (live): AC 2 scenario vault.sell-all GREEN on a lab slot. closecards3: status set to Done, merged on main; only the live or lab boxes above remain open (listed in Remaining)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sell all (N) button, two-tap in-place confirm, scope all_sellable listed by the bot, idempotent order id, tests, docs, screenshots
- [ ] #2 Scenario vault.sell-all GREEN on a lab slot (live check, not run by the worker)
<!-- AC:END -->

---
id: TASK-1052
title: 'charmvault: charms flow into the Vault, player Carry and Stow'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T18:34:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 510000
card: "charmvault"
lane: next
owns:
  - "internal/charms"
  - "internal/vaultkeep"
  - "internal/lootrules"
  - "internal/town/stash.go"
  - "internal/watch"
  - "cmd/d2bot/vaultequip.go"
  - "docs/charms.md"
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
- [ ] #1 Charms are recorded into the Vault like other items under loot.vault (big charms without the charmroom need)
- [ ] #2 Vault Carry moves an identified charm into the backpack under the keep mark, Stow moves it back; UI buttons
- [ ] #3 Sell all and the pack chores never touch a carried or pinned charm
- [ ] #4 Charm sizes and legality are data; two-build tests
<!-- AC:END -->

---
id: TASK-1052
title: 'charmvault: charms flow into the Vault, player Carry and Stow'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T18:34:00Z'
updated_date: '2026-11-27T19:42:00Z'
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
  - "cmd/d2bot/vaultkeep.go"
  - "docs/charms.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Charms already reach the Vault under loot.vault (test added). Player Carry and Stow (keeper, routes, vault page), keep-marked charms skip the stash chore, charm sizes and legality are data (internal/charms). Live checklist in docs/charms.md. closecards3: status set to Done, merged on main and every acceptance box ticked."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user decision 2026-11-27, charms are handled like any other item; the bot records them, the player carries them. Done when: Carry and Stow work offline in two builds and docs/charms.md lists the live checks.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Charms are recorded into the Vault like other items under loot.vault (big charms without the charmroom need)
- [x] #2 Vault Carry moves an identified charm into the backpack under the keep mark, Stow moves it back; UI buttons
- [x] #3 Sell all and the pack chores never touch a carried or pinned charm
- [x] #4 Charm sizes and legality are data; two-build tests
<!-- AC:END -->

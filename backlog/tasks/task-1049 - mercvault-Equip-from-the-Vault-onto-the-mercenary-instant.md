---
id: TASK-1049
title: 'mercvault: Equip from the Vault onto the mercenary (instant)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T18:24:00Z'
updated_date: '2026-11-27T19:03:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 510000
card: "mercvault"
lane: next
owns:
  - "internal/d2dctl/direct_merc.go"
  - "internal/d2dctl/direct_merc_test.go"
  - "internal/vaultkeep/equipmerc.go"
  - "internal/vaultkeep/equipmerc_test.go"
  - "internal/vaultkeep/equipdirect.go"
  - "internal/watch/vaultapi.go"
  - "internal/watch/vaultmerc_test.go"
  - "internal/watch/web/vault.js"
  - "internal/watch/web/vault.json"
  - "internal/watch/web/vault-logic.js"
  - "cmd/d2bot/vaultequip.go"
  - "cmd/d2bot/vaultkeep.go"
  - "docs/vault.md"
exclusive: []
contracts: []
result: "Built offline, unit-tested for two classes (fakes). NOT verified live: needs the fork build with item merc-equip on the sandbox image."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Equip on merc: a legal vault item is created in the pack, given to the mercenary through item merc-equip, the old piece becomes a vault record
- [x] #2 Illegal merc type, requirements, dead or absent merc, old image: refused with a reason, nothing moved
- [x] #3 Unit tests for two classes; fork/live check listed
<!-- AC:END -->

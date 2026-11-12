---
id: TASK-339
title: 'stashactions: move items between pack and stash, identify in place, deposit and withdraw gold'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T20:26:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 225000
card: "stashactions"
owns:
  - "internal/invdesk"
  - "internal/town/playerstash.go"
  - "internal/town/playerstash_test.go"
  - "internal/bot/playeraction.go"
  - "internal/bot/playeraction_test.go"
  - "cmd/d2bot/invwire.go"
  - "internal/watch/invapi.go"
  - "internal/watch/invapi_test.go"
  - "internal/watch/invui.go"
  - "internal/watch/web/inventory-actions.js"
  - "internal/watch/web/inventory-actions.css"
  - "internal/watch/testdata/itemactions"
  - "docs/inventory-actions.md"
  - "docs/whyline.md"
exclusive: []
contracts: []
result: "Built offline, NOT verified live: stash moves, in-place identify of stash items, gold deposit and withdraw as town errands sharing one stash visit; fakes and node goldens for Paladin and Sorceress."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Move to stash and Take out on the item menu, queued as town errands
- [x] #2 Identify of a stash item runs in place in the stash (no take-out)
- [x] #3 Deposit and withdraw gold with an amount box
- [x] #4 One stash visit for all queued stash jobs; trip home from the field
- [x] #5 Tests on Paladin and Sorceress with fakes, node goldens, API route tests
- [x] #6 Docs and live-check list in docs/inventory-actions.md
<!-- AC:END -->

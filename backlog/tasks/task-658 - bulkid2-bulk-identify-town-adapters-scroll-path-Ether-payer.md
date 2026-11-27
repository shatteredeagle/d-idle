---
id: TASK-658
title: 'bulkid2: bulk identify town adapters, scroll path, Ether payer'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-19T14:12:00Z'
updated_date: '2026-11-19T14:53:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 326000
card: "bulkid2"
owns:
  - "internal/bulkid"
  - "internal/town/bulkid.go"
  - "internal/town/bulkid_need_test.go"
  - "internal/town/chores.go"
  - "internal/town/stash.go"
  - "internal/town/town.go"
  - "internal/vaultkeep"
  - "internal/watch/vaultapi.go"
  - "cmd/d2bot/bulkidwire.go"
  - "cmd/d2bot/vaultkeep.go"
  - "cmd/d2bot/invwire.go"
  - "cmd/d2bot/d2dwire.go"
  - "cmd/d2bot/d2dbackend.go"
  - "docs/design/bulk-identify-2026-11-18.md"
exclusive: []
contracts: []
result: "Town adapters (Mover, Lister, CainTalk), the bulkid chore after identify, file journal with recovery under any policy, Cain quest word gate, scroll path (policy 2, buys only with town.identifyTome) and an Ether tap payer built offline with must-fail tests. NOT run live: the Cain drill is design sections 9 and 11. Ether charge semantics (what replaces the identify tool) is U. closecards3 Remaining (live): #3 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Town adapters (Mover, Lister, CainTalk) and the bulkid chore exist; journal recovery runs under any policy
- [x] #2 Scroll path and Ether tap payer built with must-fail tests (floor, cap, short, off)
- [ ] #3 Live Cain drill (design sections 9 and 11) run
<!-- AC:END -->

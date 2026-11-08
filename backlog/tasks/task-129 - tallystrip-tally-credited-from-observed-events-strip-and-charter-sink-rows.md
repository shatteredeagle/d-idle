---
id: TASK-129
title: 'tallystrip: tally credited from observed events, strip and charter sink rows'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:26:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies: []
priority: high
ordinal: 121000
card: "tallystrip"
owns:
  - "internal/currency/tallyrows.go"
  - "internal/currency/tallyrows_test.go"
  - "internal/watch/tallystrip.go"
  - "docs/tally-strip.md"
exclusive: []
contracts: []
result: "Built offline, fixture-tested, NOT verified live. Credits, charter quote and spend, strip view and hook done (docs/tally-strip.md). AC 2 open: not mounted in serve.go/web/bridge, no order slot 2 or perk. Extra files: internal/currency/tallyrows.json, registry.json sources, watch tallystrip_test.go and testdata."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 item 1 (the tally ledger side is un-paused as part of the loop; the Raven's Mark item stays deferred), fun-ideas.md section 13 card 3 and section 3.4. Kill: the user cannot say what tally is for after the first session. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Ledger credits from Hub events through registry source rows (kills per 25, quest, waypoint when the event exists (U), first, boss, completed expedition); prices are registry rows
- [ ] #2 Strip on the watch header and the return screen; a spend sheet listing the next three buys (charter, order slot 2, first perk) with ledger spend; sinks wired: order slot 2 (MaxOpenOrders from a ledger-read value) and one perk that is an existing knob row
- [x] #3 docs/tally-strip.md says built vs verified live; go vet and go test -p 1 of the touched packages; leakcheck 0
<!-- AC:END -->

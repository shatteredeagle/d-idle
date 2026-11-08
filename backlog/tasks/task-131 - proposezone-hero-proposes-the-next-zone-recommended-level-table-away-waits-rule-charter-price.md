---
id: TASK-131
title: 'proposezone: hero proposes the next zone (recommended-level table, away-waits rule, charter price)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:26:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies:
  - TASK-129
priority: medium
ordinal: 123000
card: "proposezone"
owns:
  - "internal/decide/proposezone.go"
  - "internal/decide/proposezone.json"
  - "internal/decide/proposezone_test.go"
  - "internal/watch/proposals.go"
  - "internal/watch/proposals_test.go"
  - "internal/watch/testdata/proposals/"
  - "docs/proposezone.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 item 2 and 3 of the decisions, fun-ideas.md section 13 card 5; folds in TASK-115 qqueue (nothing fades for the player, proposals queue for the return). Kill: the user never accepts a proposal and always orders from the map. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Proposal point with the zone range table (SME values kept as data, UNSURE entries tagged); when the player is away the hero waits and keeps farming the current safe zone, the proposal queues on the return screen; no advance without the player's yes, no in-range or timed auto-advance
- [ ] #2 Above-range orders from the map cost a charter in tally only, with the hero's warning; the guard keeps the last word on arrival; two-build test
- [x] #3 docs/proposezone.md says built vs verified live; go vet and go test -p 1 of the touched packages; leakcheck 0
<!-- AC:END -->

result: offline fixtures only, not verified live, not wired into cmd/d2bot (docs/proposezone.md). AC 2 stays open: the charter price for map orders was built by TASK-129; the proposal accept path is tested (402 above range) but the hero warning for map orders is unchanged. Edits outside owns: internal/watch/funwire.go, returnscreen.go (SectionButton.Act, proposal button), funwire_test.go (rig option).

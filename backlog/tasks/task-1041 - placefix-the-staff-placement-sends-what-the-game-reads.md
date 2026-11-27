---
id: TASK-1041
title: 'placefix: the staff placement sends what the game reads'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T09:22:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 503000
card: "placefix"
lane: next
owns:
  - "internal/questboard/"
  - "internal/bot/questboard.go"
  - "data/questboard/act2.json"
  - "tools/provingground/scenarios/stepexec-act2.json"
  - "tools/provingground/scenarios/questdrill.json"
  - "internal/scenario/placefix_test.go"
  - "docs/questboard.md"
exclusive: []
contracts: []
result: "Duriel drills set record 10 to 0x0c78 (bit 0 meant 'Staff already placed' to d2d, which silently dropped the placement); seven_tombs opens on the Staff made (new quest key requiresCleared) instead of the Staff quest done. Not build-specific. Offline tests only; NOT verified live (rerun pending)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: drill.act2.duriel.stepexec blocked 3/3: `cmd place 105 25` replied ok-empty and the Staff stayed in the pack. The verb and its ids are right (npcs index, held item id; fork apps/d2d/devctl/verbs.hpp). The drill SETUP wrote record 10 = 1; d2d's SevenTombsQuest::join reads bit 0 as the Staff already placed, so World::object_item returns without a reply. With the honest word (0x0c78) the board locked seven_tombs behind horadric_staff done, which only the placement sets. Done when: setups keep bit 0 clear and seven_tombs opens on the Staff made.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Duriel drill setups keep record 10 bit 0 clear (d2d reads bit 0 as the Staff already placed and drops C->S 0x44)
- [x] #2 seven_tombs unlocks on the Staff made (horadric_staff cleared), not on horadric_staff done, which only the placement sets
- [x] #3 must-fail tests for both
- [ ] #4 drill.act2.duriel.stepexec rerun places the Staff (live, coordinator)
<!-- AC:END -->

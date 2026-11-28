---
id: TASK-1131
title: 'fix14: review 14 fixes: kit skip at boss levels, hole hold, merc equip journal, counters'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T09:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 536000
card: "fix14"
lane: next
owns:
  - "internal/admit"
  - "internal/quest/objective.go"
  - "internal/bot"
  - "internal/combat"
  - "internal/vaultkeep"
  - "internal/arbiter"
  - "internal/town"
exclusive: []
contracts: []
result: "Review 14 H1 to H5, M1 and M5 fixed with tests that fail on the old code (fakes; two classes where the class matters); nothing verified live. H3: the on-road kit skip keeps the potion half for a road area a quest handler row works in (quest.ObjectiveAreas, derived from the handler steps, no area list); portal in hand is a trip, none is a hold (that the trip rules then walk the hero out is U). H5: Clear and Kill select the profile again before refusing (the rest came with the merged fix8b). H4: a hole hold waits in town instead of asking for a fresh game. H1: a takeback journal step; a gone copy is given only when the hand shows the piece in the mercenary slots (optional MercWearer; the bot hand in cmd/d2bot does not implement it yet, so there an unanswered give keeps the vault record and raises a notice: a possible second copy, never a loss). H2: at most the one new piece of the same merc slot is filed, never a piece the hand reports worn. M1: separate decline and failure counters. M5: transitive edges before routing. Open: M2, M3, M4 (not started) and M6 (ops repo, not owned). Nothing build-specific; no new knob."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: review 14 (docs/design/fable-review-2026-11-28/14-round3-merged-since-midnight.md) found five high and six medium defects in code merged since midnight. Done when: each has a test that fails on the old code and passes now.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 H3: on-road kit skip keeps the potion half at a road's boss or chest level (both judges, two classes)
- [x] #2 H5: Clear and Kill select the profile again before refusing (two classes)
- [x] #3 H4: a hole hold waits in town, no fresh game (two classes)
- [x] #4 H1: merc refusal with a lost take-back reply keeps the vault record; gone copy is given only with the slot bytes
- [x] #5 H2: merc repair files at most the displaced piece of the same slot, never a worn hero item
- [x] #6 M1: arbiter declines and failures have separate counters
- [x] #7 M5: a reinserted town chore respects every edge (property test)
- [ ] #8 M2, M3, M4, M6 of review 14
<!-- AC:END -->

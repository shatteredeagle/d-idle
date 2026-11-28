---
id: TASK-1114
title: >-
  fix1b: item flows judged by bytes not unit id, held rule, merc key per
  attempt, legacy park note
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T05:42:00Z'
updated_date: '2026-11-28T08:06:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 534000
card: "fix1b"
lane: next
owns:
  - "internal/vaultkeep"
  - "internal/questboard"
  - "internal/d2dctl"
exclusive: []
contracts: []
result: "NOT verified live (offline fakes only, Sorceress and Paladin; merc test on two merc kinds). N2: resolveDirect judges an unanswered direct Equip by the bytes in the slot (old bytes: nothing moved; the new item's bytes, stored in the order as New: swapped), never by unit id; an unreadable occupied slot keeps the order open; an empty slot now holding the new item retires its record. N1: carried charms are matched to units by bytes in one pass, found units are written back, and the carry repair compares the count of equal-bytes items with the count before the move (Seen), so a twin is not taken for the charm. N3: heldBy (active equip order, carry or stow intent, merc intent) is honoured by saleSkip, Carry, the direct Equip precheck and EquipMerc. N5: the merc create key is the attempt's order id. B2: ResumeNote on the legacy board says only the player clears the park; legacy Reset clears stored parks. Build-specific part: none. Left: backpack-only matching needs a location in the item list (U); the settle time for a command the engine runs after the look (review N2 last point), N4 two-hander and journal write errors, N6, N8, B1, B3, B4. U live: the fork reports a worn item with the norm bytes it was created from."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fable review round 2 part A (docs/design/fable-review-2026-11-28/10-fix-review-items-and-board.md) findings N2, N1, N3, N5, B2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 N2: an open direct Equip order is judged by item bytes, right across a renumbering restart (must-fail test)
- [x] #2 N1: Stow and the carry repair match carried charms by bytes (whole hero, the item list has no location), 2 and 3 charms across a restart (must-fail test)
- [x] #3 N3: Sell, Carry and Equip on mercenary refuse a record or worn item under an open order or intent (must-fail test)
- [x] #4 N5: a merc Equip retried after a refusal works; never created twice after success (must-fail test)
- [x] #5 B2: legacy board: ResumeNote truthful, Reset clears a stored park (must-fail test)
- [ ] #6 Live check: direct Equip, Carry, Stow across a new game on a throwaway hero
<!-- AC:END -->

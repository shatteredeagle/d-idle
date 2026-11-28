---
id: TASK-1114
title: 'fix1b: item flows judged by bytes not unit id, held rule, merc key per attempt, legacy park note'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:42:00Z'
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fable review round 2 part A (docs/design/fable-review-2026-11-28/10-fix-review-items-and-board.md) findings N2, N1, N3, N5, B2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 N2: an open direct Equip order is judged by item bytes, right across a renumbering restart (must-fail test)
- [ ] #2 N1: Stow and the carry repair match carried charms by bytes in the backpack, 2 and 3 charms across a restart (must-fail test)
- [ ] #3 N3: Sell, Carry and Equip on mercenary refuse a record or worn item under an open order or intent (must-fail test)
- [ ] #4 N5: a merc Equip retried after a refusal works; never created twice after success (must-fail test)
- [ ] #5 B2: legacy board: ResumeNote truthful, Reset clears a stored park (must-fail test)
- [ ] #6 Live check: direct Equip, Carry, Stow across a new game on a throwaway hero
<!-- AC:END -->

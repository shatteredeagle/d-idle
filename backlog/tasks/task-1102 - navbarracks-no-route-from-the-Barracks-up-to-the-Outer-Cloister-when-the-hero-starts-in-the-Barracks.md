---
id: TASK-1102
title: 'navbarracks: no route from the Barracks up to the Outer Cloister when the hero starts in the Barracks'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:18:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 526000
card: "navbarracks"
lane: next
owns:
  - "internal/nav/"
  - "internal/d2dworld/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

nav.barracks-return RED in all four knob arms (off, grid, gridwarp, all), identical: 'd2dworld: arrived in level 28: 0 adjacent entries, 1 levels not linked yet', then 44 x 'action failed travel to Outer Cloister: no route from Barracks (28) to Outer Cloister (27): nav: no exit to that level', failsafe ladder to rung 4 and pause within 15 s wall. With nav.warpRead 1 the warptriggers verb was read 18 times and the route was still missing. Side finding in the same log: the town portal click is dropped 3 times ('d2d portal: no object after the click'). U whether the fork or the bot loses the up exit. Run dir provingground-out/k3-000613. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run). Blocks the nav.fullGrid and nav.warpRead ledger rows (5 areas).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 nav.barracks-return GREEN for Paladin seeds 11 and 3 with the knobs off and with nav.fullGrid, nav.warpRead, quests.layoutRead on
- [ ] #2 A unit test covers a field start in level 28 with 0 adjacent entries
<!-- AC:END -->

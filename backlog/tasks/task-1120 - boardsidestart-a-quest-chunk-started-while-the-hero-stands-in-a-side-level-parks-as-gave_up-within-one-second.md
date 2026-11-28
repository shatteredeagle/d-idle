---
id: TASK-1120
title: 'boardsidestart: a quest chunk started while the hero stands in a side level parks as gave_up within one second'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T06:19:00Z'
labels:
  - bot
  - bug
dependencies:
  - TASK-1102
priority: medium
ordinal: 533000
card: "boardsidestart"
lane: next
owns:
  - "internal/questboard/"
  - "internal/bot/questboard.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

board.sm.restart-mid-chain Paladin 11 (lab rerun 2026-11-28, bot f30206f0d, run dir provingground-out/20261128-010108 of the labops2 lab checkout): the levelling plan had walked the hero into the Forgotten Tower ('d2dworld: arrived in level 20: 0 adjacent entries'); the tapped chunk search_for_cain/scroll started at 01:01:32, its step 'Go to the Dark Wood' failed three times between 01:01:33.0 and 01:01:33.9 with 'exit to Black Marsh: nav: no exit to that level: level 6 from 20', and the board logged 'parked search_for_cain/scroll ... gave up after 3 tries'. After the restart the park was kept (correct), so the chunk never ran. Root of the missing exit is the same as TASK-1102 (a level entered with 0 adjacent entries); this card is the board side: three identical failures in under a second are not three tries, and the chunk has a road through town. Evidence: docs/probes/lab-night-results-2026-11-28b.md. CHEATED setup (mechanics only), not live-verified beyond the lab run. Done when the criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A chunk whose travel step fails with 'nav: no exit to that level' falls back to town (portal or waypoint) or waits and retries on a later tick, instead of three tries inside one second
- [ ] #2 Unit test: hero in a level with 0 adjacent entries at chunk start, two classes, the chunk is not parked gave_up
- [ ] #3 board.sm.restart-mid-chain GREEN for Paladin 11 after the fix (lab run)
<!-- AC:END -->

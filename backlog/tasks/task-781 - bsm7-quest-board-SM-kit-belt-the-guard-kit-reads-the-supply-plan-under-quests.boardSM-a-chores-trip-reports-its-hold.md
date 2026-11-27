---
id: TASK-781
title: >-
  bsm7: quest board SM kit-belt (the guard kit reads the supply plan under
  quests.boardSM, a chores trip reports its hold)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T17:46:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 360000
card: "bsm7"
owns:
  - "internal/task/executor.go"
  - "internal/task/gate_hold_test.go"
  - "internal/bot/routeguard.go"
  - "internal/bot/questboard.go"
  - "internal/bot/kitplan_test.go"
  - "tools/provingground/scenarios/board.json"
  - "docs/probes/boardsm-progress-2026-11-21.md"
exclusive: []
contracts: []
result: "With quests.boardSM the guards and the shop read the supply plan; a coded fault that inserts the chores trip reports its hold (Result.Via, log line quest board: hold needs_potions), and a clean trip that leaves the shortage is a hold, not a counted failure. Unit tested (Paladin and Sorceress worlds), bot suite green. Live (second worker, CHEATED setups): hold.restock Paladin 11 shows the hold line, no park and the road onward; restock-stash-gold Paladin 11 withdrew from the stash with no park; Sorceress runs were cut by the lab pace (BLOCKED, not judged); broke, too-weak and timeout not run yet. closecards3 Remaining (live): #3 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 with quests.boardSM the guards and the shop read the supply plan (belt plus pack, minima clamped to the belt) so a chores trip can close the kit
- [x] #2 a coded fault that inserts the chores trip reports its hold (Result.Via) and the board logs 'quest board: hold needs_potions' with nothing counted
- [ ] #3 board.hold.* scenarios green live (the second worker)
<!-- AC:END -->

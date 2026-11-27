---
id: TASK-856
title: 'qdmatrix: quest drill matrix for the chunks after Tools of the Trade'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-23T05:09:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 387000
card: "qdmatrix"
owns:
  - "docs/probes/quest-drill-matrix-2026-11-23.md"
  - "tools/provingground/scenarios/questdrill.json"
  - "internal/power/archetype.go"
  - "internal/power/skilldamage_test.go"
  - "internal/questboard/state.go"
  - "internal/questboard/state_test.go"
  - "internal/d2dsrc/items.go"
  - "internal/d2dsrc/srv_test.go"
  - "internal/nav/nav.go"
  - "internal/task/act2.go"
  - "internal/task/act2_test.go"
  - "data/questboard/act2.json"
  - "internal/nav/act2hints.go"
  - "internal/nav/act2hints_test.go"
  - "internal/nav/exit_test.go"
  - "internal/task/act4.go"
  - "internal/bot/questboard.go"
  - "internal/bot/questboard_s7b_test.go"
  - "tools/probes/presets.py"
  - "tools/emu/am_batch.sh"
  - "tools/emu/am_model.py"
  - "internal/scenario/scenario.go"
  - "internal/scenario/setup.go"
  - "internal/scenario/setup_test.go"
  - "internal/saves/progress.go"
  - "tools/probes/objectlist.py"
  - "cmd/d2bot/task_test.go"
  - "internal/nav/exit.go"
  - "internal/nav/gatedoor.go"
  - "internal/nav/gatedoor_test.go"
  - "internal/nav/exitgoal_test.go"
  - "internal/nav/mouthgate.go"
  - "internal/bot/goal.go"
  - "internal/bot/goal_acts_test.go"
  - "cmd/d2bot/task.go"
  - "cmd/d2bot/acts_test.go"
  - "internal/talk/talk.go"
  - "internal/talk/talk_test.go"
exclusive: []
contracts: []
result: "Quest drill matrix through Act 2 (docs/probes/quest-drill-matrix-2026-11-23.md). Lab verdicts (CHEATED setups, mechanics only, not verified on the live hero): Sail east GREEN (Paladin, Sorceress), Radament GREEN, Horadric Cube GREEN, Assemble GREEN to Cain, Andariel held by readiness (correct for the kit); Tainted Sun, Maggot Lair shaft, Summoner, Duriel RED with named nav causes. Bot fixes with must-fail tests: attack pick by paid points, act hop unlock and dialog row, proxy done never beats a lock (Duriel and Hellforge done on an Act 1 hero), exit and door walks carry a goal, door wait, GoToUnit door, Deckard Cain class by act, Tainted Sun word, earlier act not reachable, waypoint when no walk, Seven Tombs canyon steps, pre.stats and pre.waypoints -1."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Matrix doc lists every chunk through Act 2 with a verdict
- [x] #2 Each RED has a root cause or a named blocker
<!-- AC:END -->

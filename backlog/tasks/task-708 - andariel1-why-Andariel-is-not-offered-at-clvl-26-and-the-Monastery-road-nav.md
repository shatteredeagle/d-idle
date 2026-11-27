---
id: TASK-708
title: 'andariel1: why Andariel is not offered at clvl 26, and the Monastery road nav'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T02:53:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 340000
card: "andariel1"
owns:
  - "internal/questboard/desk.go"
  - "cmd/d2scenario/tap.go"
  - "internal/nav/exit.go"
  - "tools/provingground/scenarios/act1more.json"
  - "internal/d2map/tracker.go"
  - "internal/d2dworld/world.go"
  - "internal/d2dworld/world_test.go"
  - "internal/nav/exit_test.go"
  - "internal/questboard/desk_test.go"
  - "cmd/d2scenario/tap_test.go"
exclusive: []
contracts: []
result: "F3: Andariel is held back by the readiness model (no merc or dead merc, short of potions, would likely lose: too hard at clvl 26 without spent skills), correct and kept; the board row now says why and the tapper logs it (tests on two builds). F4: NOT a missing exit in the data: at the Monastery Gate d2dworld has 5 adjacent entries, pending 1, linked 27, but no border cell and no walkable strip toward 27 (run 20261121-000708-4eec log 'nav no usable gate'); nav now waits for late borders, walks to unproven strips, ignores a stale no-exit mark (must-fail tests) and logs the facts. Scenario act1.andariel.slaughter stays RED on 'Entered Outer Cloister' (Paladin 11 and Sorceress 3, CHEATED setups, lab runs 20261120-230516-0e0d, -232851-6a3a, 20261121-000708-4eec). Next: dump the layout strip of level 26 against 27 (debug layout/nearby) and fix the geometry in internal/d2dworld/border.go."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: act1more finding F3/F4 (docs/probes/provingground-act1more-2026-11-20.md): Andariel never offered at clvl 26 and the Monastery Gate to Outer Cloister exit not found. Done when: the board says why, the nav waits for a late border, and the scenario runs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Board row says why Andariel is held (test, two builds)
- [x] #2 Tapper logs why a chunk is not startable
- [x] #3 Nav: waits for a late border and walks to an unproven strip (must-fail tests; the Monastery Gate case itself is NOT fixed, see result)
- [ ] #4 Scenario act1.andariel.slaughter GREEN or the reason asserted, Paladin 11 and Sorceress 3
<!-- AC:END -->

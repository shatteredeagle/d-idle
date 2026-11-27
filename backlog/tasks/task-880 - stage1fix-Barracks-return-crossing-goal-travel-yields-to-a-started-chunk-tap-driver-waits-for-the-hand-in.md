---
id: TASK-880
title: >-
  stage1fix: Barracks return crossing, goal travel yields to a started chunk,
  tap driver waits for the hand-in
status: Done
assignee:
  - credit-runner
created_date: '2026-11-23T09:02:00Z'
updated_date: '2026-11-27T23:46:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 395000
card: "stage1fix"
owns:
  - "internal/nav/exit.go"
  - "internal/nav/exit_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/questboard.go"
  - "internal/bot/questboard_test.go"
  - "internal/bot/denreward_test.go"
  - "cmd/d2scenario/tap.go"
  - "cmd/d2scenario/tap_test.go"
  - "tools/provingground/scenarios/navexit.json"
  - "tools/provingground/scenarios/necro.json"
  - "docs/probes/necromancer-act1-drill-2026-11-22.md"
exclusive: []
contracts: []
result: "Three fixes with offline must-fail tests (nav strip crossing, goal travel yields to a started chunk, tap driver waits for the hand-in); stage 1 GREEN in the lab afterwards (drill doc section 12: the fixture had no mana potion, so the kit gate sent the hero to town between clear and hand-in; CHEATED setup, mechanics only). Open: den hand-in after a town trip loses the reward (separate card); nav.barracks-return scenario pending (hero start cannot reproduce the arrival). denreward follow-up (2026-11-27): the pending-reward hold now covers every non-emergency trip (pack full, rebuy, merc) and the guard, must-fail test on Sorceress and Paladin (TestPendingRewardHoldsEveryNonEmergencyTrip), offline only; nav.barracks-return authored with a walked-arrival warp start (validate ok, not run live). Earlier: three fixes with offline must-fail tests (nav strip crossing, goal travel yields to a started chunk, tap driver waits for the hand-in); stage 1 still RED at the Den hand-in in the stage-1 setup (Den alone GREEN at clvl 10, 14, 16): cause open; nav.barracks-return scenario pending (hero start cannot reproduce the arrival). Not ready to play."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Unproven strip crossing with a must-fail test on two classes
- [x] #2 Goal travel yields once to a runnable chunk, tested on two classes
- [x] #3 Tap driver waits for the hand-in, tested
- [x] #4 Stage 1 GREEN (lab, CHEATED setup: docs/probes/necromancer-act1-drill-2026-11-22.md section 12, job 20261123-041955-81fd; not verified on a played hero)
<!-- AC:END -->

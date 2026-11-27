---
id: TASK-1070
title: 'fieldstate: hand the return leg to the fighter when hostiles are near'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T21:55:00Z'
updated_date: '2026-11-27T22:46:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 515000
card: "fieldstate"
lane: next
owns:
  - "internal/bot/loop.go"
  - "internal/bot/goal.go"
  - "internal/bot/state.go"
  - "internal/bot/runner.go"
  - "internal/bot/fieldstate_test.go"
exclusive: []
contracts: []
result: "Not verified live (offline). Cause (V code read, incident files not in tree): the return leg (Returning to the field, StateResupplying) runs travel/walk fallbacks with no fighter and without the walking flag; the guard only reacts to drink/chicken verdicts there. Fix: a choreWatch on the leg, guard cancels with errFieldCrowd after choreGrace (2 s, existing const) when a hostile is within combat.fightRadius, do() ends the trip and the next step fights. Build-independent (place rule); test runs Sorceress and Paladin. None of hostilespin, admit2, arbloop, akarafix touch this state."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Hostiles near on the way back to the field end the leg within the grace and hand the hero to the fighter
- [x] #2 Must-fail test with two classes, virtual time
<!-- AC:END -->

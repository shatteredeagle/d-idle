---
id: TASK-36
title: 't0fight: engineered drill for a fight-time question and a learned answer'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-06T16:02:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - drill
  - ladder
  - T0
  - parked
dependencies: []
priority: high
ordinal: 36000
card: "t0fight"
owns:
  - "docs/probes/t0fight.md"
  - "internal/drill/t0fight.go"
  - "internal/drill/t0fight_test.go"
  - "cmd/d2drill/t0fight_lab.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Closes TASK-21 AC 2 and 3 for the fight points: docs/probes/t0drill.md says live acceptance is met for the town points only. A fresh clvl 1 hero cannot reach an elite or low life by itself, so the setup writes before runstart.

Done when: Drill sets up low life or an elite pack by SETUP writes before d2cheat runstart; no gameplay write after...; A fight-time question opens mid-fight and is answered in the field; the hero keeps fighting while it is...; A second run after a learned answer behaves differently (rulesLearned > 0 and the action changes) on two... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Drill sets up low life or an elite pack by SETUP writes before d2cheat runstart; no gameplay write after the start; record says 'writes after start: 0'
- [ ] #2 A fight-time question opens mid-fight and is answered in the field; the hero keeps fighting while it is open (no pause); log lines show thought and answer
- [ ] #3 A second run after a learned answer behaves differently (rulesLearned > 0 and the action changes) on two builds; result says verified live or not
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parked in the 2026-11-10 true-up: ladder mechanism waits for the re-scope of TASK-19.
<!-- SECTION:NOTES:END -->

---
id: TASK-1033
title: 'necroteeth: the Necromancer casts its main skill, not plain Attack'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T06:15:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 500000
card: "necroteeth"
lane: next
owns:
  - "internal/combat/classdata/profiles.json"
  - "internal/combat/necro_test.go"
  - "internal/leveler/mainattack_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cause found offline: lab prep or bot (row logs read)
- [ ] #2 Main attack of the default plan is learned by clvl 12 when the bot spends the banked points (test on two builds)
- [ ] #3 Necromancer profiles drop the Lower Resist when magicResist curse row (Lower Resist does not lower magic resist); must-fail test
- [ ] #4 Prep fix for the featuresoak rows written in the report
- [ ] #5 Live: a clvl 12 Necromancer row with points spent casts Teeth
<!-- AC:END -->

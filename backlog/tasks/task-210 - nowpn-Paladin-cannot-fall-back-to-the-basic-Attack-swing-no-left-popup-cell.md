---
id: TASK-210
title: 'nowpn: Paladin cannot fall back to the basic Attack swing (no left popup cell)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T13:50:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 163000
card: "nowpn"
owns:
  - "internal/combat/bindleft.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: live Paladin (clvl 10, mana 0) logged "left skill not selectable Attack: skill 0 has no left popup cell" on every swing, so it never meleed with its Scepter and backed off for mana (kills 0.2/min). The d2d dump lists only trained skills plus the selected one, so Attack is absent from Player.Skills and leftCells found no cell. Done when: leftCells synthesizes Attack. result: fixed in internal/combat/bindleft.go, offline test only (fails before, passes after); not yet verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Attack (skill 0) is found in the left popup although the d2d dump omits it from Player.Skills
- [x] #2 Unit test fails before, passes after
<!-- AC:END -->

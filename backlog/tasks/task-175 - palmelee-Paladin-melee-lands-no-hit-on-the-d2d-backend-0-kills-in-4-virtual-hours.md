---
id: TASK-175
title: 'palmelee: Paladin melee lands no hit on the d2d backend (0 kills in 4+ virtual hours)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:59:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: high
ordinal: 142000
card: "palmelee"
owns:
  - "docs/palmelee.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by paladinsmoke (TASK-142, docs/paladinsmoke.md). A fresh Paladin (lockstep d2d, lab verbs off, basic attack because no skill is learned before clvl 2) made 0 kills, 0 XP in 4+ virtual hours in Blood Moor: log 'combat no bound attack skill, swinging the basic attack' known=0, 'ignoring target ... casts 15 life 100% engaged 22s', 'approaches' ignores at 10 subtiles; the kill-drought rule then leaves the game every 3 virtual minutes (64 re-entries in 15 wall minutes). Hypothesis (U): fight.go says a melee click walks the hero to the target by itself (true for game.exe); d2d's cmd skill 0 with a unit may only swing in reach and not walk. Done when: a live drill shows a fresh Paladin kills Blood Moor monsters on d2d (kills per minute, clvl 2 reached), with writes after start 0, and an offline test for the chosen fix with two classes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Root cause named (bot walk vs d2d verb), tagged V from a live probe
- [ ] #2 A fresh Paladin reaches clvl 2 in Blood Moor on lockstep d2d, writes after start: 0
- [ ] #3 Offline test with Paladin and Barbarian profiles
<!-- AC:END -->

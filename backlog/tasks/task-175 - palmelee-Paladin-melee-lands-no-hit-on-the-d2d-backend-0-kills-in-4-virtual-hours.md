---
id: TASK-175
title: 'palmelee: Paladin lands no hit in lockstep d2d (0 kills in 11 virtual hours); realtime kills slowly'
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
  - "internal/d2dcombat/input.go"
  - "internal/d2dcombat/input_test.go"
  - "internal/combat/fighter.go"
  - "internal/combat/swing_test.go"
  - "docs/probes/paladinsmoke/run-b.sh"
exclusive: []
contracts: []
result: "Root cause V: bare d2d click without a hover never attacked; Swing by unit id fixes it. Lockstep drill: clvl 9 / 1195 kills in 240 virtual minutes, writes after start 0 (docs/palmelee.md). Verified live yes."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by paladinsmoke (TASK-142, docs/paladinsmoke.md). A fresh Paladin (lockstep d2d, lab verbs off, basic attack because no skill is learned before clvl 2) made 0 kills, 0 XP in 4+ virtual hours in Blood Moor: log 'combat no bound attack skill, swinging the basic attack' known=0, 'ignoring target ... casts 15 life 100% engaged 22s', 'approaches' ignores at 10 subtiles; the kill-drought rule then leaves the game every 3 virtual minutes (64 re-entries in 15 wall minutes). Control (V, same bot commit, same class, same fork binary, container in realtime): the Paladin made 13 kills and 315 XP in 20 real minutes (then slow again), so the failure is lockstep-specific or much worse in lockstep. Hypothesis (U): in lockstep a melee click is one command per step and the hero never gets the ticks to walk and swing; fight.go says a melee click walks the hero to the target by itself (true for game.exe and, slowly, realtime d2d). Done when: a live drill shows a fresh Paladin kills Blood Moor monsters on lockstep d2d (kills per minute, clvl 2 reached), with writes after start 0, and an offline test for the chosen fix with two classes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Root cause named (bot walk vs d2d verb), tagged V from a live probe
- [x] #2 A fresh Paladin reaches clvl 2 in Blood Moor on lockstep d2d, writes after start: 0
- [x] #3 Offline test with Paladin and Barbarian profiles
<!-- AC:END -->

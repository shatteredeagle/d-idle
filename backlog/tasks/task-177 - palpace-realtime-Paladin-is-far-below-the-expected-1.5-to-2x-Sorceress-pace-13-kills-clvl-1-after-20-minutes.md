---
id: TASK-177
title: 'palpace: realtime Paladin is far below the expected 1.5 to 2x Sorceress pace (13 kills, clvl 1 after 20 minutes)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T04:19:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies:
  - TASK-175
priority: medium
ordinal: 144000
card: "palpace"
owns:
  - "docs/palpace.md"
exclusive: []
contracts: []
result: "Realtime after the fix: 62 kills, clvl 2 in 20 min (before 13 kills, clvl 1); remaining idle = recovery roaming in a depleted Blood Moor; expectation corrected (docs/palmelee.md, docs/palpace.md). Verified live yes, writes after start 0."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by paladinsmoke (TASK-142, docs/paladinsmoke.md). Container, realtime, fresh Paladin, first 20 minutes: 13 kills, 315 of 500 XP, clvl 1, gold 7, 0 deaths, hero stays in Recovery: farming gold in Blood Moor with long idle stretches (no kill between minute 6 and 10). docs/research/class-early-game.md expects clvl 5 to 6 in about 20 minutes (U). Done when: a diagnosis names why the melee loop idles in realtime (target loss, approach, drought, potion trips) and a realtime run reaches clvl 3 inside 20 minutes or the expectation is corrected with a V number.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Idle stretches explained from the minute lines and combat decisions
- [x] #2 Realtime Paladin run of 20 minutes with the fix, kills and clvl recorded
<!-- AC:END -->

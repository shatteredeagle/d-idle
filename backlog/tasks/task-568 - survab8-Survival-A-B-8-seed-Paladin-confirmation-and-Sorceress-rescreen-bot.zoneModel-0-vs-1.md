---
id: TASK-568
title: >-
  survab8: Survival A/B 8-seed Paladin confirmation and Sorceress rescreen,
  bot.zoneModel 0 vs 1
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T21:56:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - probe
dependencies: []
priority: medium
ordinal: 296000
card: "survab8"
owns:
  - "docs/probes/survival-ab-8seed-2026-11-16.md"
  - "docs/probes/test-characters.md"
exclusive: []
contracts: []
result: "Paladin 8 seeds: abreport FAIL (XP ratio 0.823, interval wholly below zero) though chickens 10 vs 36 and deaths 1 vs 2; recommendation: not yet. Sorceress rescreen skipped (gate failed). Lockstep d2d only, writes after start 0 in 16 of 16, not verified in game.exe. closecards3: #3 not ticked, moot: the Paladin gate failed (result), so no Sorceress rescreen."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user's plan needs 8 seeds before the bot.zoneModel default flips. Done when: the verdict doc is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Paladin 8 seeds x 2 arms x 2 h run on lockstep slots, writes after start 0
- [x] #2 abreport verdict with slack 1 and the metric tables
- [ ] #3 Sorceress 3-seed rescreen if the Paladin gate passes
- [x] #4 Plain recommendation for the bot.zoneModel default flip
<!-- AC:END -->

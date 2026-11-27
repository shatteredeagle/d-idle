---
id: TASK-497
title: 'botstuck: Dark Wood tree search and Forgotten Tower stairs stall'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-15T14:37:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 276000
card: "botstuck"
owns:
  - "internal/nav/mouthgate.go"
  - "internal/nav/walk.go"
  - "internal/nav/fixmouth4_test.go"
  - "cmd/d2bot/objects.go"
  - "internal/bot/staletarget.go"
  - "internal/bot/task.go"
  - "internal/bot/parkstep_test.go"
  - "internal/env/tuning_bot.go"
  - "internal/task/executor.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. nav: the mouth gate skips an unwanted exit within mouthTriggerR of a wanted exit (test TestGateSkipsMouthBesideWantedExit). d2bot objectAdapter.find explores up to 100 s until the object or its preset appears. Park rule: internal/bot/staletarget.go parkStep/unparkDue, test parkstep_test.go (class neutral, reads step ids and knobs only). No unit test for the find explore path: it needs a real nav.Navigator whose simulator lives in package nav (unexported). Knobs are bot.parkAfter and bot.parkMinutes (flat keys like the other bot knobs, not bot.stuck.*). closecards3 Remaining (live): #3 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the sandbox hero (clvl 15) loops: the Inifuss tree step fails 'object class 30 not found' within 1 s of Dark Wood arrival (12 times in 3 h; d2d lists presets of built rooms only, nothing explores), and in the Forgotten Tower the mouth gate refuses every click to the cellar stairs (they lie inside the Black Marsh mouth's 15-subtile disc), 12 minutes standing twice. Done when: both fixed offline with tests; live check ticked later.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tree of Inifuss step explores the level until the tree or its preset shows up instead of failing in 1 s
- [x] #2 Mouth gate no longer refuses clicks toward a wanted exit that lies inside an unwanted exit's disc
- [ ] #3 Live check: Doofer reaches the Inifuss tree and the Tower Cellar stairs
- [x] #4 After 2 identical not-found planning blocks of one quest step the step is parked 15 min (knobs bot.parkAfter, bot.parkMinutes), the level plan is farmed, the reason shows as journeyBlocked, a different step is not parked
<!-- AC:END -->

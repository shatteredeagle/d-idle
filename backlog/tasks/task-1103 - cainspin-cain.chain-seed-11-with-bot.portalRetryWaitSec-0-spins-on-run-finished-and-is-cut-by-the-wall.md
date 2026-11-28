---
id: TASK-1103
title: 'cainspin: cain.chain seed 11 with bot.portalRetryWaitSec 0 spins on ''run finished'' and is cut by the wall'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:19:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 527000
card: "cainspin"
lane: next
owns:
  - "internal/bot/"
exclusive: []
contracts: []
result: "Cause (V, bot log of run 20261127-232846): after the Tristram failsafe ladder a flail hold (bot.flailHoldMin 5) started at 23:30:14; the hero then walked to Tower Cellar 3, cleared it, and flailHold kept calling sweep on the cleared area. Each pass (nav mouth gate holds the click, roam done, area cleared, run finished) takes no virtual time, so lockstep virtual time stood at 1719 s (pace lines, steps 14177) and the hold never ended: class A (the hold and the sweep undo each other, neither waits). The portalRetryWaitSec 0 arm only changed the road into the hold (cancelled portal, chicken, relaunches). Fix: flailHold waits 5 s in a cleared area with nothing hostile; a sweep pass over an area already cleared that kills nothing takes one tick (covers the area-done gap and board hold paths too). Must-fail tests Paladin and Sorceress (TestFlailHoldInClearedAreaLetsTimePass, TestSweepPassOnClearedAreaTakesATick). Not build-specific. NOT verified live: #2 needs a rerun of cain.chain Paladin 11 with the knob."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Run 20261127-232846: 2108 s wall, bot log 2.7 million lines, BLOCKED (cut before its virtual budget). The last 200000 lines are a cycle of 'roam: done', 'area cleared', 'sweep: coverage N/N cleared=true', 'run finished', 'gold ledger' (28569 each) and 'd2d: leave: re-entered the game' from line 1485 on. Seeds 12 and 13 without the knob were GREEN (344 s and 228 s). Also in this job: the determinism pair of seed 11 differed hard (rep1 RED quest_bits record 4 got 0x0008, rep2 GREEN; first divergence BOT-SIDE at tick 3974, an extra 'debug presets' read), which is the TASK-680 flake, not settled by this night. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The cause of the spin is named from the log (which loop logs roam done, area cleared, run finished without advancing virtual time)
- [ ] #2 cain.chain Paladin 11 with bot.portalRetryWaitSec 0 ends inside its virtual budget
<!-- AC:END -->

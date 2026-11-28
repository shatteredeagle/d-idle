---
id: TASK-1107
title: 'act2night: three Act 2 stepexec drill findings (Duriel stall, shaft road without a portal, potion not drunk in the cube drill)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:19:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 531000
card: "act2night"
lane: next
owns:
  - "internal/bot/"
  - "tools/provingground/scenarios/stepexec-act2chunks.json"
exclusive: []
contracts: []
result: "shaft.stepexec (not verified live): the hero spent its only portal scroll escaping a nav stall at the Maggot Lair 1 exit to level 63, walked back in, and inside the lair the early road guard refused the next step for the missing portal; the chores trip it inserted could not reach town, the chunk parked after 3 tries in 0.3 s and the bot idled (Waiting for a fresh game) until the virtual budget. Fix: the early road kit is an entry rule, a hero already on the road goes on (internal/bot/earlyroute.go, test TestEarlyRoadKitIsAnEntryRule, Paladin and Sorceress); level and life floors still close it. msf is mapped in questItemCodes (V code). nf3b (not verified live): duriel.stepexec 3 of 3 runs: Duriel died, then the talk_tyrael row failed 'npc not found or not reachable' (seen=false: Tyrael is not in the snapshot from the arena), the row's 2 tries and the chunk's 3 were spent in 0.2 s, the chunk parked and the bot idled. Fix: the executor's Approach walks the level for a field NPC out of sight before the talk (internal/bot/stepexec.go, test TestTalkRowLooksForAFieldNPC); U that the walk reaches his chamber on d2d. cube.stepexec potion-not-drunk: explained: the board went back and forth between two refused steps (cube road 'area banned after repeated deaths', level.23 'no eligible reachable leveling area') in the Sewers with 8 hostiles; a refused step runs no action, so no guard asked the safety check for about 100 s of game time (life 99 to 18 percent, first drink when an action ran). Fix: the failure path asks the safety check before it waits (internal/bot/loop.go fail, test TestFailedStepAsksTheSafetyCheck, Paladin and Sorceress). Nothing build-specific. Open: place_staff ran once in town after a pocket escape (recovered on retry); why both cube steps were refused at once (the area ban) is not looked at."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

duriel.stepexec 0/3: 'quest board: starting chunk seven_tombs/duriel', then 'Waiting for a fresh game: no eligible area is left in this game', 'campaign stalled: fresh game failed: attached d2d cannot be restarted', exit after 60 virtual min (harness cannot restart; the bot found no way to the chunk). shaft.stepexec RED: 'action failed to town: no town portal available', 'horadric_staff/shaft: not ready: early road (no town portal for the act2_maggot_lair road)', item msf never picked (setup gap or product: the hero cannot buy a scroll from the field start). cube.stepexec RED on 2 hits of invariant potion-not-drunk (a survival finding). Blocks the quests.stepExec ledger row (Duriel 2 of 3). Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 drill.act2.duriel.stepexec: the bot plays (no 'campaign stalled' exit) in 2 of 3 runs
- [x] #2 drill.act2.shaft.stepexec: the setup gives the hero a town portal or the road guard does not need one
- [x] #3 potion-not-drunk in drill.act2.cube.stepexec explained (life 35 percent under the 50 percent line for 18 s with 4 Super Healing in the belt, drank 0)
<!-- AC:END -->

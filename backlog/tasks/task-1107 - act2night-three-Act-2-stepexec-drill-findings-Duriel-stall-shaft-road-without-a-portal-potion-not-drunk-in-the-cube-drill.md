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
result: "shaft.stepexec (not verified live): the hero spent its only portal scroll escaping a nav stall at the Maggot Lair 1 exit to level 63, walked back in, and inside the lair the early road guard refused the next step for the missing portal; the chores trip it inserted could not reach town, the chunk parked after 3 tries in 0.3 s and the bot idled (Waiting for a fresh game) until the virtual budget. Fix: the early road kit is an entry rule, a hero already on the road goes on (internal/bot/earlyroute.go, test TestEarlyRoadKitIsAnEntryRule, Paladin and Sorceress); level and life floors still close it. msf is mapped in questItemCodes (V code). Duriel stall and cube potion-not-drunk not addressed here."
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
- [ ] #3 potion-not-drunk in drill.act2.cube.stepexec explained (life 35 percent under the 50 percent line for 18 s with 4 Super Healing in the belt, drank 0)
<!-- AC:END -->

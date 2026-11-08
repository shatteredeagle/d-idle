---
id: TASK-183
title: 'recoverypace: every class sits in Blood Moor on the recovery gold farm for most of the first 90 minutes, so class pace is not measured'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-08T11:07:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: medium
ordinal: 152000
card: "recoverypace"
owns:
  - "docs/soaks/recoverypace.md"
  - "docs/probes/recoverypace/"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/playertuning_test.go"
  - "internal/resource/recovery.go"
  - "internal/resource/recovery_test.go"
  - "internal/resource/tuning.go"
  - "internal/env/tuning_resource.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Partly done. Diagnosis (V): recovery is 19 to 30 of 90 minutes, not 68; it ends late (20 min stall, 60 min hold) although the kit is affordable. Fix: resource.recoveryEndOnKit (code default 0) plus a player/tester preset (ladder on only: stall 10 min / 50 gold, povertySellMin 8). Verified live on Paladin and Sorceress, one seed per arm, 75 to 87 virtual minutes: recovery 19->11 and 23->10 min, town 15->8 and 15->5 min; XP at minute 75 -26 and +49 percent (noise, U). Barbarian clvl 6 not re-run, so AC1 is not ticked. docs/soaks/recoverypace.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Evidence: docs/probes/classpacing/minutes/bar1.jsonl (68 of 90 minutes in area 2, clvl 4 at minute 90, 238 kills, step 'Recovery: farming gold') and 'area skipped dry-recovery 900s' in the assassin run; same finding as follow-up 1 of docs/palmelee.md. Done when: the criterion holds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Barbarian lockstep run reaches clvl 6 within 90 virtual minutes or the card result names why not; the recovery farm leaves a dry area before 15 minutes
<!-- AC:END -->

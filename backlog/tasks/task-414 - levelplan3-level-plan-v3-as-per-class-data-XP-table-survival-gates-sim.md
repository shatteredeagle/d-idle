---
id: TASK-414
title: 'levelplan3: level plan v3 as per-class data (XP table, survival gates, sim)'
status: In Progress
assignee:
  - pro-runner
created_date: '2026-11-14T11:59:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 251000
card: "levelplan3"
owns:
  - "internal/levelplan"
  - "internal/plandata/levelplan-normal.json"
  - "docs/plan/data/levelplan-normal.json"
  - "docs/levelplan3.md"
  - "internal/plandata/boss_levels.go"
  - "internal/bot/plan.go"
  - "internal/bot/levelplan3_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/resource.go"
  - "internal/plandata/plandata_test.go"
  - "internal/plandata/boss_levels_test.go"
  - "data/campaign/normal.json"
  - "internal/campaign/campaign_test.go"
  - "internal/task/guided_test.go"
  - "internal/boss/types.go"
  - "internal/boss/run.go"
  - "internal/boss/boss_test.go"
  - "internal/boss/fake_test.go"
  - "internal/combat/fighter.go"
  - "cmd/d2bot/boss.go"
  - "cmd/bossrun/adapter.go"
  - "docs/levelplan.md"
  - "internal/watch/testdata/campaign"
exclusive: []
contracts: []
result: "Level plan v3 as data (levelplan-normal.json, internal/levelplan): Paladin active (Blood Moor 1-3, Cold Plains 3-6, Stony Field 5-8, Caves 6-9, Black Marsh 7-11, Dark Wood 8-11, Tamoe 9-13, Jail 1 11-14, Catacombs 2/3 12-18, merc +2 from Tamoe; quests named only), Barbarian and Sorceress review rows; bot Pick uses it for the Paladin. Predicted (I) 20.4k XP clvl 6 at 60 min, 63.8k clvl 10 at 120, 170.6k clvl 14 at 240 (plan v2 12.9k / 27.7k on the same model). B1 melee no-kite; Smith journey 14/16; Paladin Andariel 20/18 (floors 17/15). Build-specific: the Paladin slice table (data). Offline only, NOT verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

User go 2026-11-14: rewrite the leveling plan guided by kolbot-SoloPlay, the AREAMON XP table and the boss/hitrec survival limits. Done when the plan is data, tested offline and wired for the Paladin.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Level plan v3 data: ordered farm and quest slices per class (Paladin active; Barbarian, Sorceress rows), entry/exit clvl, areas, way in, merc and kit gates
- [x] #2 Validity tests: area ids exist, bands monotone, no gap (with and without merc), entry < exit, quest slices name spine chapters, unknown class errors
- [x] #3 Simulation over the AREAMON XP table: predicted XP at 60/120/240 min; old plan reproduces the AREAMON model within 15 percent
- [x] #4 Bot Pick uses the active class plan (Paladin); other classes keep progression-normal.json
- [x] #5 Boss findings: B1 melee heroes do not kite Andariel; B2/B3 Smith and Andariel floors in data; B4 Countess listed for the task owner
- [x] #6 docs/levelplan3.md with V/S/I/U tags and kolbot deviations
- [ ] #7 Live check: one measured fresh Paladin hour against the predicted 60-minute XP
<!-- AC:END -->

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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

User go 2026-11-14: rewrite the leveling plan guided by kolbot-SoloPlay, the AREAMON XP table and the boss/hitrec survival limits. Done when the plan is data, tested offline and wired for the Paladin.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Level plan v3 data: ordered farm and quest slices per class (Paladin active; Barbarian, Sorceress rows), entry/exit clvl, areas, way in, merc and kit gates
- [ ] #2 Validity tests: area ids exist, bands monotone, no gap (with and without merc), entry < exit, quest slices name spine chapters, unknown class errors
- [ ] #3 Simulation over the AREAMON XP table: predicted XP at 60/120/240 min; old plan reproduces the AREAMON model within 15 percent
- [ ] #4 Bot Pick uses the active class plan (Paladin); other classes keep progression-normal.json
- [ ] #5 Boss findings: B1 melee heroes do not kite Andariel; B2-B4 Smith, Andariel, Countess floors in data
- [ ] #6 docs/levelplan3.md with V/S/I/U tags and kolbot deviations
- [ ] #7 Live check: one measured fresh Paladin hour against the predicted 60-minute XP
<!-- AC:END -->

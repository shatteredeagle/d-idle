---
id: TASK-368
title: 'hourfix: first-hour time sinks (bounce ban, idle clear, Den claim, Cold Plains order, junk pickup)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T16:33:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 237000
card: "hourfix"
owns:
  - "docs/hourfix.md"
  - "internal/bot/loop.go"
  - "internal/bot/goal.go"
  - "internal/bot/leveling.go"
  - "internal/bot/task.go"
  - "internal/bot/lifelog.go"
  - "internal/bot/runner.go"
  - "internal/plandata/progression-normal.json"
  - "docs/plan/data/progression-normal.json"
  - "internal/env/tuning_bot.go"
  - "internal/env/tuning_loot.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fixes from docs/probes/hour1-analysis-2026-11-12.md. Done when each fix has a test and docs/hourfix.md tags it V/U.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bounce skip takes the cave, not the transit area
- [ ] #2 bot.clearIdleMin ends an idle leveling field
- [ ] #3 a claimed Den is no quest run again
- [ ] #4 melee Cold Plains at clvl 4, Den first
- [ ] #5 poorMinSell 20
- [ ] #6 whole-repo gate, race, checks green
<!-- AC:END -->

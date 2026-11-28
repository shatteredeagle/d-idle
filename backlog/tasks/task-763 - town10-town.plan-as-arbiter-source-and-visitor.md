---
id: TASK-763
title: 'town10: town.plan as arbiter source and visitor'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-21T04:53:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 350000
card: "town10"
owns:
  - "internal/arbiter"
  - "internal/supply"
exclusive: []
contracts: []
lane: next
result: "offline only, not verified live: internal/arbiter/upkeep.go: one upkeep source (SrcUpkeep, family chores) fed by PlanFacts(supply.Plan), UpkeepSources() = StdSources with trip and resupply collapsed into it; it is NOT in StdSources, so the shadow and every default path are unchanged. Two builds (mana need 2 and 0) in tests. AC2 is a precondition, not met: bot.arbiterControl is 0 and the default mask 3 leaves the chores family in shadow; the bot-side wiring (arbfam in internal/bot, merging PlanFacts into the View, retiring the old trip deciders in internal/task and internal/bot) is outside this card owns and waits for the flip. Build-specific: none in Go. Remaining (live): AC 2 precondition, the arbiter must leave shadow for the town family (bot.arbiterControl is 0); bot-side wiring waits for the flip."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 one upkeep intent replaces trip and resupply; the three trip deciders collapse to one
- [ ] #2 starts only after the arbiter leaves shadow for the town family
<!-- AC:END -->

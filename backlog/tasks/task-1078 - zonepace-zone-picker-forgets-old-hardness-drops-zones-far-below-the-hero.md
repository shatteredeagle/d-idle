---
id: TASK-1078
title: 'zonepace: zone picker forgets old hardness, drops zones far below the hero'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T23:24:00Z'
updated_date: '2026-11-27T23:50:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 517000
card: "zonepace"
lane: next
owns:
  - "internal/survival/"
  - "cmd/d2bot/survivalwire.go"
exclusive: []
contracts: []
result: "lockHoldMinutes (480) and bandDrop (normal 6, nightmare 8, hell 8) in survival params.json; no new knob (acts only under dark bot.zoneModel 1); offline tests, both builds; NOT verified live. Hardness decay and xp/min-ranked pick with the unvisited discount already existed."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: Questy farmed Dark Wood at clvl 15-16 for 5-6k XP/h (docs/probes/sandbox-log-analysis-2026-11-27.md problem 2). Hardness already decays (hardHalfLifeMinutes 120, V ledger.go hardNow; the file shows the raw product); what never lapsed was the left-for-cost lock (only a 10 percent power change lifted it), and no rule dropped zones far below the hero. Done when: both rules are data in params.json, inside the dark bot.zoneModel (no new knob), tested for two builds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A left-for-cost lock lapses after lockHoldMinutes (data); test fails without the fix
- [x] #2 Zones more than bandDrop[diff] levels above their monster level leave the candidates (data), never emptying the list
- [x] #3 Both builds (paladin, sorceress) tested with a fake clock
- [ ] #4 Live Questy check: picks Black Marsh or Tamoe at clvl 15-16 (needs live sandbox)
<!-- AC:END -->

---
id: TASK-1129
title: 'zone5: zone model supply cap rows, caster opens in a field not a graveyard'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T08:16:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 535000
card: "zone5"
lane: next
owns:
  - "internal/survival/"
  - "cmd/d2bot/survivalwire.go"
  - "docs/survival/zonemodel.md"
exclusive: []
contracts: []
result: "supplyMinutes rows in survival params.json (caster 20; new rows summoner, ranged_physical, hybrid_tank_caster; bruiser left as measured): the zone prior is capped by the units the zone holds, so the caster no longer opens in Burial Grounds, Crypt or Mausoleum; k parts diagnostic in the deciding zone choice line; no new knob (acts only under dark bot.zoneModel 1). Build-specific part: none in Go, the rows are per archetype in params.json. The k clamp and merc-left-dead suspects of holdout 5 are not isolated offline. Offline tests, Sorceress and Paladin; NOT verified live (lab checklist in docs/survival/zonemodel.md section 8)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The Sorceress arm of the bot.zoneModel A/B fails (ratio 0.497): the model opens in Burial Grounds at a prior the zone cannot supply. Done when the offline simulation shows the field pick and the lab checklist is written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sorceress and Paladin zone-choice simulation: the caster opens in a field with supply, the bruiser run is unchanged; fails without the rows
- [x] #2 Every power archetype has a survival row (summoner, ranged_physical, hybrid_tank_caster added)
- [x] #3 k parts diagnostic in the deciding zone choice line
- [ ] #4 Live: 3-seed Sorceress A/B screen per docs/survival/zonemodel.md section 8 (live check)
<!-- AC:END -->

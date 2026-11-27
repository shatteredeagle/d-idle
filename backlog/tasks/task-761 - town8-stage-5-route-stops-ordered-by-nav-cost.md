---
id: TASK-761
title: 'town8: stage 5 route, stops ordered by nav cost'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:52:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 348000
card: "town8"
owns:
  - "internal/supply/"
  - "internal/town/"
exclusive: []
contracts: []
result: "offline only, not verified live: stage 5 (town.plan >= 5) orders the enabled chores by the cheapest walk over act-table role positions (NPC and stash presets read at run time) under precedence rules (internal/town/route.go: identify and equip before sell, sell before buy, paid chores keep their spend order, one stash stop that settles the purse first only when the purse cannot pay, else last); deterministic, falls back to the fixed list on missing presets or an edge cycle; a log line town plan: route stops=... cost=...; scenarios one-visit and determinism already exist in townplan.json (d2scenario validate ok), live run pending. Distance is straight-line between preset positions, not nav path length (U). Build-specific: none, roles come from town.Acts. Knob text in internal/env/tuning_town.go still says route not built (outside this card owns)."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 stops ordered under the precedence rules; fixed list retired behind the knob
- [x] #2 scenarios one-visit, determinism
<!-- AC:END -->

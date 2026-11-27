---
id: TASK-759
title: 'town7: short-gold priority table as data and handover'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:52:00Z'
updated_date: '2026-11-27T19:42:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 347000
card: "town7"
owns:
  - "internal/supply/"
  - "internal/town/"
  - "internal/resource/"
exclusive: []
contracts: []
result: "offline only, not verified live: priority table is data (internal/supply/priority.json, per-profile override supported, validated by a test), Solve reads it; supply.Plan.Handover names the gold sources (sell, stash, vault prompt, recovery) (the guard already holds with needs_gold_farm, never attention; wiring the sources into internal/task and internal/bot is outside this card owns); the stage 3 plan line gains sell=, dropped= and hand-over fields for the scenarios gold-short-priorities and short-gold-handover (already in townplan.json, d2scenario validate ok, not run live). Build-specific: none in Go; lines come from profiles.json and the belt plan. Two builds in tests. closecards3: status set to Done, merged on main and every acceptance box ticked."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 priority table in data; short-gold hands over to sell, stash, vault prompt, recovery; no needs_attention for gold
- [x] #2 scenarios gold-short-priorities, short-gold-handover
<!-- AC:END -->

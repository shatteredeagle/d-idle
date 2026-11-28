---
id: TASK-762
title: 'town9: viewer, GET /api/townplan and the Town plan card'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:53:00Z'
labels:
  - watch
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 349000
card: "town9"
owns:
  - "internal/watch"
  - "internal/supply"
exclusive: []
contracts: []
lane: next
result: "offline only, not verified live: GET /api/townplan (watch.townplan/1, internal/watch/townplanapi.go, 503 until bound, POST refused) from supply.ViewOf (internal/supply/view.go: lines, budget, stops, waiting sentence, minute line); Town plan card as opt-in panel web/panels/town-plan (text helpers tested in node on recorded plans of two builds, Sorceress-like mana need 2 and Paladin-like 0). Left: cmd/d2bot must call TownPlanAPI.Bind with the last plan of Town.SupplyPlan and set Fun.TownPlan (outside owns; not wired, so the route is absent today), and the layout that places the opt-in panel. Build-specific: none in Go."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GET /api/townplan schema watch.townplan/1, additive
- [x] #2 Town plan card, waiting sentence, minute line; UI test on recorded plans
<!-- AC:END -->

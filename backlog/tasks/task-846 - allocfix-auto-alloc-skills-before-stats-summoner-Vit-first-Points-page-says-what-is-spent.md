---
id: TASK-846
title: >-
  allocfix: auto-alloc skills before stats, summoner Vit-first, Points page says
  what is spent
status: Done
assignee:
  - credit-runner
created_date: '2026-11-23T03:43:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 384000
card: "allocfix"
owns:
  - "cmd/d2bot/autoallocwire.go"
  - "cmd/d2bot/autoallocwire_test.go"
  - "internal/points"
  - "internal/plandata/build-necromancer_summoner_normal30.json"
  - "docs/plan/data/build-necromancer_summoner_normal30.json"
  - "internal/watch/pointsui.go"
  - "internal/watch/web/points.js"
  - "docs/autoalloc.md"
  - "tools/provingground/scenarios/autoalloc.json"
exclusive: []
contracts: []
result: "Test bed read-only: the allocator was not dead, it spent stats first at calm moments only (10 Energy from the old rows, then Vit; first skill 2 min after the lock). Fixed: skills before stats, summoner rows Vit-first, Points page shows unspent/spent/waiting. Unit tests two classes, -race x20 ok. Scenario points.autoalloc-necro-clvl4 written, NOT run (builder long lane busy); not verified live. closecards3 Remaining (live): #4 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Skills are planned before stats; test names two classes
- [x] #2 Summoner table rows 2-3 are Vitality first
- [x] #3 Points page shows unspent, spent and waiting
- [ ] #4 Scenario points.autoalloc-necro-clvl4 run on the builder (live check)
<!-- AC:END -->

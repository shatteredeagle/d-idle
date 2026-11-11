---
id: TASK-303
title: 'repairtrip: Repair now becomes a repair trip (safe moment, portal or walk home, smith, back) and Farm until repair'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T15:52:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: high
ordinal: 211000
card: "repairtrip"
owns:
  - "docs/repair.md"
  - "docs/live-checks.md"
  - "internal/repair"
  - "internal/goldfarm"
  - "internal/bot/repairtrip.go"
  - "internal/bot/repairtrip_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/goldfarm.go"
  - "internal/bot/runner.go"
  - "internal/bot/whyline.go"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "internal/town/repairtrip_test.go"
  - "internal/watch/repairapi.go"
  - "internal/watch/repairapi_test.go"
  - "internal/watch/web/panels/gear"
  - "internal/watch/web/inventory.js"
  - "internal/watch/web/gear.js"
  - "internal/watch/testdata/gear_repairtrip_dom.js"
  - "internal/watch/testdata/gear_repairtrip_dom.txt"
  - "internal/watch/testdata/gear_farm_dom.txt"
  - "internal/watch/testdata/compat"
  - "internal/watch/whyline.json"
  - "internal/watch/whyline_test.go"
  - "cmd/d2bot/repairwire.go"
  - "cmd/d2bot/goldfarmwire_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: user decision 2026-11-11, manual repair becomes a trip. Done when: docs/repair.md live checklist holds live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Button state from live pack gold vs estimate x1.3 (Repair now / Farm until repair / disabled with reason), additive watch.repair schema 2
- [x] #2 Repair trip: safe moment, loot, portal only when clear, scroll-less walk home, smith, back through own portal, in-town shortcut, failure keeps request open
- [x] #3 Farm until repair: farm order to repairNeed x1.3 + kit + margin, then trip, then ends itself; progress line
- [x] #4 Guards: cooldown, max 2 trips, auto-cancel, cancel button, persisted, one active request
- [x] #5 UI: panel and inventory card states, DOM goldens, compat fixture, uilint; Paladin and Sorceress tests
- [ ] #6 Whole-repo gate on the worker host, merge queued
<!-- AC:END -->

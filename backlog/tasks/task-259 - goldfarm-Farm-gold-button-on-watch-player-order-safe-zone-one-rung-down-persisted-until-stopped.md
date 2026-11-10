---
id: TASK-259
title: >-
  goldfarm: Farm gold button on /watch (player order, safe zone one rung down,
  persisted until stopped)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T17:27:00Z'
updated_date: '2026-11-10T18:18:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: high
ordinal: 195000
card: "goldfarm"
owns:
  - "docs/goldfarm.md"
  - "internal/goldfarm"
  - "internal/bot/goldfarm.go"
  - "internal/bot/goldfarm_test.go"
  - "internal/watch/goldfarmapi.go"
  - "internal/watch/goldfarmapi_test.go"
  - "internal/watch/web/panels/gear"
  - "internal/watch/testdata/gear_farm_dom.js"
  - "internal/watch/testdata/gear_farm_dom.txt"
  - "cmd/d2bot/repairwire.go"
  - "cmd/d2bot/goldfarmwire_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/broke.go"
  - "internal/bot/runner.go"
  - "internal/bot/whyline.go"
  - "internal/watch/funwire.go"
  - "internal/watch/whyline.json"
  - "internal/watch/whyline_test.go"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/ladder.go"
exclusive: []
contracts: []
result: "Built and tested offline only (fixtures, stub DOM); nothing verified live. Design and open choices in docs/goldfarm.md. check.py fails only on the duplicate palprofile id (task-251 and task-254) from main."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the sandbox poverty loop (a poor hero cycling the same zones); the player wants one button that farms gold in a safe zone until told to stop. Done when: docs/goldfarm.md acceptance holds live (checklist there).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Farm gold button and status line on the gear panel, POST/GET /api/farmgold, persisted state
- [x] #2 Safe zone one rung down as plan data, Sorceress and Paladin tests
- [x] #3 Why line, log line, area-thrash and poverty interaction documented and tested offline
- [x] #4 Whole-repo gate on the worker host, merge queued
<!-- AC:END -->

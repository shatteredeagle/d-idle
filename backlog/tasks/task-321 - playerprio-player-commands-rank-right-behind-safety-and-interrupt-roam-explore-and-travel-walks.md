---
id: TASK-321
title: >-
  playerprio: player commands rank right behind safety and interrupt roam,
  explore and travel walks
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T19:07:00Z'
updated_date: '2026-11-11T19:54:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 219000
card: "playerprio"
owns:
  - "internal/bot/playeraction.go"
  - "internal/bot/playeraction_test.go"
  - "internal/bot/loop.go"
  - "internal/bot/state.go"
  - "internal/bot/runner.go"
  - "internal/town/playerid.go"
  - "docs/inventory-actions.md"
exclusive: []
contracts: []
result: "Built offline, NOT verified live: guard interrupt (errPlayer) for player commands and repair now, hero-walking condition removed, tests on fakes."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: S

Why: live sandbox: queued identifies never ran because the hero is always walking. Done when: tests pass; nothing verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Guard interrupts roam/explore/travel walks for a queued player command (identify, equip, trash, errands, repair now) at a safe moment; walking no longer blocks it
- [x] #2 Safety first, fight holds the command, clean abort and requeue, Paladin and Sorceress tests, docs ladder
<!-- AC:END -->

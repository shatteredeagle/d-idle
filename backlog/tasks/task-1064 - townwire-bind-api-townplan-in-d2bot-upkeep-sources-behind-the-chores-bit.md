---
id: TASK-1064
title: 'townwire: bind /api/townplan in d2bot, upkeep sources behind the chores bit'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T21:12:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 513000
card: "townwire"
lane: next
owns:
  - "docs/knobs.md"
  - "cmd/d2bot/townplanwire.go"
  - "cmd/d2bot/townplanwire_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "internal/bot/shadow.go"
  - "internal/bot/upkeepwire_test.go"
  - "internal/town/planshadow.go"
  - "internal/town/planview_test.go"
  - "internal/town/town.go"
  - "internal/town/route.go"
  - "docs/arbiter.md"
  - "backlog/tasks/task-765 - town12-SME-review-of-the-supply-profile-numbers-advice-only.md"
exclusive: []
contracts: []
result: "route mounted in both backends (503 until bound, NoView until a visit solved a plan), PlanFacts and UpkeepSources behind bot.arbiterControl plus the chores bit (defaults byte-identical; a town without a planner keeps the old sources; V tests for a Sorceress and a Paladin); old trip deciders NOT collapsed (follow-up); Town plan card stays opt-in; not verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Wiring the town9/town10 cards could not own. Done when the route answers in both backends and the upkeep source runs behind the chores family bit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GET /api/townplan answers from the town's last plan (503 until bound); tests for two builds
- [x] #2 PlanFacts merged and UpkeepSources used only when bot.arbiterControl is on and the chores bit is set; defaults unchanged
- [x] #3 town12 card path fixed (internal/supply/profiles.json)
- [x] #4 whole-repo vet and short tests run on the worker host
<!-- AC:END -->

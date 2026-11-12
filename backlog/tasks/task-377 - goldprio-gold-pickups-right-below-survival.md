---
id: TASK-377
title: 'goldprio: gold pickups right below survival'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T19:53:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 241000
card: "goldprio"
owns:
  - "internal/loot"
  - "internal/bot/goldprio.go"
  - "internal/bot/goldprio_test.go"
  - "docs/goldprio.md"
  - "cmd/d2bot/adapters.go"
  - "cmd/d2bot/looteradapter_test.go"
  - "internal/combat/interlude_test.go"
  - "internal/combat/fight.go"
  - "internal/combat/types.go"
  - "internal/bot/goal.go"
  - "internal/bot/contracts.go"
  - "internal/bot/runner.go"
  - "internal/env/tuning_loot.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Cause: looterAdapter hid the loot memory (0 loot_memory lines in 56 min), no pickup during Clear, area leave skipped the walk-back, failed clicks forgot piles. Fix: Clear interlude gold grab, roam stop, gold before leaving, retries, knobs loot.goldPriority/gold*. Docs: docs/goldprio.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the bot missed gold: loot memory never ran in production (adapter hid it), no pickup mid-fight, area leave skipped the walk-back. Done when: gold is taken right below survival (docs/goldprio.md). Consumer: itself.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Loot memory and gold ledger reach the runner through the adapter (cmd/d2bot)
- [x] #2 Mid-fight gold grab (Clear interlude) below survival, with danger/hostile/life gates, knobs loot.gold*
- [x] #3 Roam stops for a pile in reach; sweep fetches remembered gold before leaving an area
- [x] #4 Failed gold clicks retried over calls (loot.goldRetries); gold exempt from the walk-back trip cap
- [x] #5 Tests: Paladin + Sorceress; docs/goldprio.md with numbers from the sandbox log
<!-- AC:END -->

---
id: TASK-191
title: 'stancelever: player-tunable fight stance row on the policy sheet'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T15:22:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: high
ordinal: 155000
card: "stancelever"
owns:
  - "internal/decide"
  - "internal/watch"
  - "internal/plandata"
  - "internal/env/tuning_bot.go"
  - "docs/stance.md"
  - "cmd/d2bot"
  - "docs/plan/data/chores-defaults.json"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Stance row Cautious/Balanced/Aggressive with per-class knobs, Paladin class start, session record line, return card text; tests for two classes and POST golden. Offline only, NOT verified live; Aggressive survival values U."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Part of the 2026-11-08 feedback (Paladin does not fight). See docs/stance.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 stance row Cautious/Balanced/Aggressive with per-class knob table
- [x] #2 tests for two classes, POST /api/policy golden, defaults unchanged
<!-- AC:END -->

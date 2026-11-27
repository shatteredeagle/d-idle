---
id: TASK-1015
title: 'necrosummoninv: summon-missing soak invariant for a Necromancer fight'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T19:13:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 486000
card: "necrosummoninv"
lane: next
owns:
  - "internal/bot/invariants.go"
  - "internal/bot/invariants_test.go"
  - "internal/bot/observe.go"
  - "internal/d2/player.go"
  - "internal/d2dsrc/source.go"
  - "internal/d2dsrc/source_test.go"
  - "cmd/d2scenario/invariants.json"
  - "docs/invariants.md"
exclusive: []
contracts: []
result: "Soft invariant summon-missing in the bot monitor (summon skill learned from the skills.txt summon column, mana in hand, hostiles near, state line pets=0 for 10 s) plus minute line minions=N; tests on two heroes and on Necromancer and Druid skill data. Not seen on a live state line (U)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 see docs/probes/necro-readiness-2026-11-26.md cards table
<!-- AC:END -->

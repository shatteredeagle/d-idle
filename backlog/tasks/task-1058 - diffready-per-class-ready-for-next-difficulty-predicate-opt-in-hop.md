---
id: TASK-1058
title: 'diffready: per-class ready-for-next-difficulty predicate (opt-in hop)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T19:46:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 512000
card: "diffready"
lane: next
owns:
  - "internal/diffready"
  - "internal/env/tuning_bot.go"
  - "internal/bot/leveling.go"
  - "internal/bot/explain.go"
  - "cmd/d2bot/util.go"
  - "data/darklaunch.json"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User decisions 2026-11-27: Hell has no resist gate, no auto retreat, hop is opt-in. Design: docs/design/difficulty-transition-2026-11-27.md section 4. Done when the criteria hold offline.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Embedded per-class data file (NM rows from SME, Hell rows U with no resist field) and a pure predicate returning a missing-reason list
- [ ] #2 Two classes in tests, unknown class or difficulty is an error naming it
- [ ] #3 bot.diffReadyV2 default-0 knob with darklaunch row; ReadyForNext and wantedDifficulty use it; the hop waits for the owner go and never happens on its own
<!-- AC:END -->

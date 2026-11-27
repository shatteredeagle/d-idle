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
knob_net: owner-ok
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
  - "internal/bot/diffready.go"
  - "internal/bot/diffready_test.go"
  - "internal/bot/runner.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "docs/diffready.md"
  - "docs/knobs.md"
exclusive: []
result: "Built and tested offline, not verified live: per-class Nightmare rows (S) and Hell rows (U, no resist field), pure predicate with missing/unread lists, knob bot.diffReadyV2 (default 0) replaces the clvl rule in wantedDifficulty and the goal chain, hop waits for Runner.GrantDifficultyGo; no operator verb calls the go yet."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User decisions 2026-11-27: Hell has no resist gate, no auto retreat, hop is opt-in. Design: docs/design/difficulty-transition-2026-11-27.md section 4. Done when the criteria hold offline.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Embedded per-class data file (NM rows from SME, Hell rows U with no resist field) and a pure predicate returning a missing-reason list
- [x] #2 Two classes in tests, unknown class or difficulty is an error naming it
- [x] #3 bot.diffReadyV2 default-0 knob with darklaunch row; ReadyForNext and wantedDifficulty use it; the hop waits for the owner go and never happens on its own
<!-- AC:END -->

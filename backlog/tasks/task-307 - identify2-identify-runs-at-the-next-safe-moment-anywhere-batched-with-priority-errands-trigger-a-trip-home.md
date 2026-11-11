---
id: TASK-307
title: 'identify2: identify runs at the next safe moment anywhere, batched, with priority; errands trigger a trip home'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T16:04:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 211000
card: "identify2"
owns:
  - "internal/invdesk"
  - "internal/bot/playeraction.go"
  - "internal/bot/playeraction_test.go"
  - "internal/town/playerid.go"
  - "internal/town/playerid_sim_test.go"
  - "internal/town/playerid_test.go"
  - "internal/watch/whyline.json"
  - "internal/watch/whyline_test.go"
  - "docs/inventory-actions.md"
  - "docs/whyline.md"
  - "internal/bot/goal.go"
  - "internal/bot/runner.go"
exclusive: []
contracts: []
result: "Built offline, NOT verified live: priority, batch, 60 s bound, field execution and errand trip tested on fakes (desk and bot)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: S

Why: user asked to bump identify priority. Done when: tests on Paladin and Sorceress fakes pass; nothing verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Identify runs at the next safe moment in town or field, ahead of equip/trash, batched in one inventory-open, 60 s bound, fails fast without a tool
- [x] #2 Town errands start a trip home at the next safe moment (shared safeMoment predicate)
<!-- AC:END -->

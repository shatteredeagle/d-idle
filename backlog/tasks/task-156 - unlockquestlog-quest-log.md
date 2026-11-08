---
id: TASK-156
title: 'unlockquestlog: quest log'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: medium
ordinal: 134000
card: "unlockquestlog"
owns:
  - "internal/watch/unlockquestlog.go"
  - "internal/watch/unlockquestlog_test.go"
  - "internal/watch/unlock.go"
  - "internal/watch/funwire.go"
  - "internal/watch/web/**"
  - "internal/watch/testdata/questlog-*.json"
  - "internal/unlock/**"
  - "data/unlocks/ladder.json"
  - "data/quests/**"
  - "docs/unlock-questlog.md"
exclusive: []
contracts: []
result: "Built and tested offline (go test ./internal/watch ./internal/unlock; golden payloads for Sorceress and Paladin); not verified live or in a browser. States come from the existing quest reader's EvQuest events (no new reader, no offsets); the live Akara read is unverified. Den reward: normal only in the data (nightmare/hell unknown, no reward line). See docs/unlock-questlog.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: Akara hands out the Den of Evil quest; list of quests with state. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: check Den of Evil quest state and reward conditions against the data before building.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Ladder row questlog opens when Akara hands out the Den of Evil quest (quest_stage trigger); locked shows a silhouette with a teaser
- [x] #2 Quests with state, current objective and why, from a data table and the existing quest events; no new reader or offset; unmet quests only counted
- [x] #3 Den of Evil state and reward checked against the data (docs/unlock-questlog.md)
- [x] #4 Golden payloads for two classes, locked/open, replay idempotency, unknown row and table errors naming them, a reveal line exists
<!-- AC:END -->

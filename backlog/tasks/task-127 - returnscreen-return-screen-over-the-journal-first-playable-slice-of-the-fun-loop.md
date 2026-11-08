---
id: TASK-127
title: 'returnscreen: return screen over the journal (first playable slice of the fun loop)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T00:26:00Z'
labels:
  - watch
  - feature
  - ladder
milestone: m-4
dependencies: []
priority: high
ordinal: 119000
card: "returnscreen"
owns:
  - "internal/watch/returnscreen.go"
  - "internal/watch/returnscreen_test.go"
  - "docs/return-screen.md"
exclusive: []
contracts: []
result: "Built offline and wired into the live bot by TASK-140 (feedHub, Serve routes, sections tally/codex/quiet). Live smoke 2026-11-07 on d2d (10 min, lockstep): /api/return filled with 4 expeditions and 90 kills; gold, waypoints, drops (autoloot off) and quests stayed empty. Trip boundary rule held on real area_change events."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 (Expedition Ledger), docs/research/fun-ideas.md section 13 card 1; first slice built on the mock of TASK-109 (design input, not a blocker). Folds in the missed-questions grouping of TASK-115. Kill: the user cannot say from the page what happened or what to do next. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 /watch lands on a return screen after an absence over 20 minutes: headline, the five moments (closest call, named boss, best drop, level-up/unlock, the stats line that moved) from existing events, expedition cards grouped by town trip from the journal and Hub events (trip boundaries are U until traced; say which events are used)
- [x] #2 Missed questions grouped by kind with 'answer once for all'; queued proposals area (empty until proposezone); three-sentence box feeding the session record (TASK-106); offline tests on a recorded journal
- [x] #3 docs/return-screen.md says built vs verified live; go vet and go test -p 1 of internal/watch; leakcheck 0
<!-- AC:END -->

---
id: TASK-604
title: 'boardmoderace: fix the data race on the task waypoint map in the quest board tests'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T09:34:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 307000
card: "boardmoderace"
owns:
  - "internal/bot/task.go"
  - "internal/bot/waypoints.go"
exclusive: []
contracts: []
lane: next
result: "AC 1 done (merged 7266196e2: mutex around task.wp and the per-game reset). Race run on the worker host, nice 19: go test -race -count=200 -cpu 1,2,4 -run Board ./internal/bot/ (75 min, 600 test runs): no DATA RACE reported. One non-race failure in that run: TestQuestBoardAcceptRunStopHandIn line 116 got Outcome confirming instead of done (the hand-in save evidence lands after the first read in the fake; a timing flake under -race load, needs its own card). AC 2 left open: not green until that flake is fixed, and the whole-package -race run was not done. Not verified live. AC 2 verified 2026-11-27 ~18:20 EST: full go test -race ./internal/bot passed in 219 s on the worker host, plus the hand-in race fixed by racefix (TASK-1076)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: S7b gate failed with DATA RACE on l.task.wp in boardmode_test (World built by the test goroutine while the runner writes). Done when: merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 l.task.wp and the per-game reset are guarded; the quest board world build off the loop goroutine no longer races
- [x] #2 go test -race -count=200 -cpu 1,2,4 on Board tests and the whole internal/bot package under -race pass
<!-- AC:END -->

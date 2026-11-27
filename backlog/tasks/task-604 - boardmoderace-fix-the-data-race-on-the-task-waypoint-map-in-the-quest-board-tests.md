---
id: TASK-604
title: 'boardmoderace: fix the data race on the task waypoint map in the quest board tests'
status: Proposed
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
result: " Remaining: AC 2 race run (-count=200 -cpu 1,2,4) was not re-run by closecards2; AC 1 code change is merged but left unticked until the race run is recorded."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: S7b gate failed with DATA RACE on l.task.wp in boardmode_test (World built by the test goroutine while the runner writes). Done when: merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 l.task.wp and the per-game reset are guarded; the quest board world build off the loop goroutine no longer races
- [ ] #2 go test -race -count=200 -cpu 1,2,4 on Board tests and the whole internal/bot package under -race pass
<!-- AC:END -->

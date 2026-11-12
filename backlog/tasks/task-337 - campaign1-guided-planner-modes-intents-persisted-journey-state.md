---
id: TASK-337
title: 'campaign1: guided planner, modes, intents, persisted journey state'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-11T20:15:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-335
priority: high
ordinal: 226000
card: "campaign1"
owns:
  - "internal/task/guided.go"
  - "internal/task/guided_test.go"
  - "internal/bot/guided.go"
  - "internal/bot/guided_test.go"
  - "internal/campaign/"
  - "internal/task/campaign.go"
  - "internal/bot/runner.go"
  - "cmd/d2bot/journeywire.go"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "docs/campaign.md"
  - "docs/inventory-actions.md"
  - "docs/quest-givers.md"
exclusive: []
result: "Guided planner (task.Campaign.WithGuide), safety walls from the floor planner, level steps name the chapter, Free/Orders modes, campaign.Controller with state/campaign.json, wired in cmd/d2bot (off in named runs). Floor offer is in the API/strip; the return-screen question queue hookup is open (AC3). Build-specific: data only. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: stage of the Guided journey (docs/campaign.md, TASK-335). Done when: the acceptance criteria below pass offline on Paladin and Sorceress.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Guided mode follows the spine: go when ready, farm with a Why naming the chapter
- [x] #2 Free and Orders modes; player orders still override the plan
- [ ] #3 level step names the chapter it unlocks, visible budget, floor offer via the question queue
- [x] #4 state/campaign.json per hero; tests on Paladin and Sorceress
<!-- AC:END -->

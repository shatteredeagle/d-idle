---
id: TASK-337
title: 'campaign1: guided planner, modes, intents, persisted journey state'
status: Proposed
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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: stage of the Guided journey (docs/campaign.md, TASK-335). Done when: the acceptance criteria below pass offline on Paladin and Sorceress.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Guided mode follows the spine: go when ready, farm with a Why naming the chapter
- [ ] #2 Free and Orders modes; player orders still override the plan
- [ ] #3 level step names the chapter it unlocks, visible budget, floor offer via the question queue
- [ ] #4 state/campaign.json per hero; tests on Paladin and Sorceress
<!-- AC:END -->

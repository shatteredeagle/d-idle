---
id: TASK-1071
title: >-
  questresume: pause policy for death parks, campaign counter under the board,
  arbiter-starved cause
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T22:00:00Z'
updated_date: '2026-11-27T22:46:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 515000
knob_net: owner-ok
card: "questresume"
lane: next
owns:
  - "internal/questboard/"
  - "data/questboard/"
  - "internal/bot/questboard.go"
  - "internal/bot/questauto.go"
  - "internal/bot/boardprogress.go"
  - "internal/bot/boardprogress_test.go"
  - "internal/bot/starvedcause.go"
  - "internal/bot/starvedcause_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/arbiter_control.go"
  - "internal/bot/shadow.go"
  - "internal/env/tuning_quests.go"
  - "internal/watch/questsection.go"
  - "internal/watch/questsection_test.go"
  - "internal/watch/funwire.go"
  - "cmd/d2bot/funwire.go"
  - "data/darklaunch.json"
  - "docs/knobs.md"
  - "docs/questboard.md"
exclusive: []
contracts: []
result: "Built and tested offline only, not verified live: pause policy (data/questboard/resume.json, knob quests.resumeAfterDeaths default 0, ledger row), paused_after_deaths reason with since and clears in /api/quests plus a Return section, campaign progress now counts board-finished quests (must-fail test), arbiter-starved names winner, area and parked runs."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: a sandbox hero did no quest for about 42 h after two deaths on one chunk (sandbox log analysis 2026-11-27, problems 1 and 7). Done when: a death park lifts under a data policy behind a dark knob, every park is visible with since and what clears it, the campaign goal counts board-finished quests, and arbiter-starved names its cause. See docs/questboard.md, section Parks after deaths.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A death park lifts by itself under a data policy behind quests.resumeAfterDeaths (default 0), never twice in a row
- [x] #2 Every park shows reason.code paused_after_deaths with since and what clears it in /api/quests and on the Return screen
- [x] #3 Campaign goal progress counts board-finished quests (must-fail test)
- [x] #4 arbiter-starved names the winner, area and parked runs
<!-- AC:END -->

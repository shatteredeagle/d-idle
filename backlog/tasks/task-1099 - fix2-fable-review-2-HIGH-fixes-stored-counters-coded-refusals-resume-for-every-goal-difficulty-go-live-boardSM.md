---
id: TASK-1099
title: 'fix2: fable review 2 HIGH fixes (stored counters, coded refusals, resume for every goal, difficulty go, live boardSM)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T03:54:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 524000
card: "fix2"
lane: next
owns:
  - "internal/questboard"
  - "internal/bot"
  - "docs/questboard.md"
exclusive: []
contracts: []
result: "Review 2 findings 1 to 7 and 10 fixed with must-fail tests on fakes (Paladin and Sorceress); not verified live. Build-specific part: none (board, loop and consent rules are class-free)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fix the HIGH findings of docs/design/fable-review-2026-11-28/02-quest-board.md with must-fail tests on fakes. Done when the per-package tests pass plain and with QUESTBOARD_SM=1 and the whole repo passes on the worker host.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Failure and death counters survive a restart with boardSM 1; the legacy board honours stored parks (must-fail tests, Paladin and Sorceress)
- [x] #2 Rung 4 skip counts only coded refusals; a stall of uncoded step failures still restarts
- [x] #3 ResumeTick runs for level, farm and idle goals
- [x] #4 The difficulty go is consumed only after the hop's new game was made
- [x] #5 Executor follows quests.boardSM live; no_path and refused_repeatedly keep the release retry; quests.board off keeps stored parks
- [x] #6 QUESTBOARD_SM=1 go test ./internal/questboard is green
- [ ] #7 Live: restart between two deaths still parks; flip, park, revert keeps the park (sandbox)
<!-- AC:END -->

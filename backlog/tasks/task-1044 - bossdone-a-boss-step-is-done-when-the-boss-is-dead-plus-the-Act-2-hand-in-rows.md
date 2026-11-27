---
id: TASK-1044
title: 'bossdone: a boss step is done when the boss is dead, plus the Act 2 hand-in rows'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T13:07:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 506000
card: "bossdone"
lane: next
owns:
  - "internal/bot/stepexec.go"
  - "internal/bot/task.go"
  - "internal/bot/bossdone_test.go"
  - "data/questspec/act2.json"
  - "data/questspec/caps"
  - "docs/design/stepexec-p9.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Lab drill act2drill5 (docs/probes/act2-stepexec-drill-2026-11-27.md): Duriel killed by the general fight, then the boss step stalls on boss not found. Done when the kill step counts a dead boss and the hand-in rows exist.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A boss kill step reads its postcondition from game state (seen kill, dead unit, or the quest word bit from data), so a boss killed earlier counts as done instead of boss not found
- [ ] #2 Act 2 hand-in rows after Duriel (Tyrael, Jerhyn, Meshif) are data rows of the existing talk kind, the last post is the record 14 completion
- [ ] #3 Must-fail tests: boss already dead is done, boss alive is fought, two builds
- [ ] #4 Live: drill.act2.duriel.stepexec passes the kill step and the hand-in on the lab
<!-- AC:END -->

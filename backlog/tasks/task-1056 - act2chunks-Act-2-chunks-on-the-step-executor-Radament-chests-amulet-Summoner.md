---
id: TASK-1056
title: 'act2chunks: Act 2 chunks on the step executor (Radament, chests, amulet, Summoner)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T19:08:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 511000
card: "act2chunks"
lane: later
epic: epic-act2
owns:
  - "data/questspec/act2.json"
  - "internal/bot/stepexec.go"
  - "internal/bot/stepexec_act2_test.go"
  - "internal/bot/stepexec_act2chunks_test.go"
  - "internal/bot/task.go"
  - "internal/questspec/"
  - "tools/provingground/scenarios/stepexec-act2chunks.json"
  - "docs/design/stepexec-p9.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Radament, cube/staff/scroll chest, amulet altar and Summoner steps are questspec rows with exec blocks
- [ ] #2 unit tests with fakes, two classes, must-fail
- [ ] #3 scenario file validates
- [ ] #4 no new knob (quests.stepExec)
<!-- AC:END -->

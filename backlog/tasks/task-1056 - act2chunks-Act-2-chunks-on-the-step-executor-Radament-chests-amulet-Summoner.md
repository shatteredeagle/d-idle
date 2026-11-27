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
  - "tools/provingground/scenarios/tiers.json"
  - "docs/design/stepexec-p9.md"
exclusive: []
result: "Rows for Radament, Cube/Staff chests, Viper altar and Summoner in data/questspec/act2.json, adapter names and no-op pickup, 6 *.stepexec scenarios (validate ok). Unit tests only, NOT verified live; the Book pick-up, Cain scroll talk and Arcane movement are not ported."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: roadmap R8, epic-act2 (TASK-937): port the next Act 2 chunks to data rows of the generic step executor (docs/design/stepexec-p9.md, section on the second batch). Done when: the rows, adapter, tests and scenarios below exist; live drills are listed, not run.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Radament, cube/staff/scroll chest, amulet altar and Summoner steps are questspec rows with exec blocks
- [x] #2 unit tests with fakes, two classes, must-fail
- [x] #3 scenario file validates
- [x] #4 no new knob (quests.stepExec)
<!-- AC:END -->

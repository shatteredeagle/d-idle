---
id: TASK-1130
title: 'flipfix2: flip tool sandbox aliases, status in EST with ledger state, graduate-list ledger check, graduation-ready note'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T08:41:00Z'
labels:
  - ops
  - bug
dependencies: []
priority: medium
ordinal: 536000
card: "flipfix2"
lane: next
owns:
  - "tools/flip/"
  - "tools/darklaunch/ledger.py"
  - "docs/process/autoflip.md"
  - "docs/process/dark-launch.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

autoflip plan could not read the sandboxes by alias; status had no remaining time; graduate-list was unchecked. Done when the four fixes are in with tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 sandbox aliases (doofer, questy) resolve from tools/flip/sandboxes.json in every flip and autoflip subcommand
- [ ] #2 flip.py status shows remaining minutes and the window end in EST plus the ledger state of each knob
- [ ] #3 autoflip graduate-list checks each row against the per-knob ledger files, with unit tests
- [ ] #4 a clean close writes graduation_note graduation-ready to the state and graduate-list prints it
- [ ] #5 live: autoflip plan reads both sandboxes after merge (watcher restart by the coordinator)
<!-- AC:END -->

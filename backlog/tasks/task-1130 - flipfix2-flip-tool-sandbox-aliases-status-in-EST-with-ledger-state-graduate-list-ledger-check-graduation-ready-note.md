---
id: TASK-1130
title: >-
  flipfix2: flip tool sandbox aliases, status in EST with ledger state,
  graduate-list ledger check, graduation-ready note
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T08:41:00Z'
updated_date: '2026-11-28T09:00:00Z'
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
result: "Aliases from tools/flip/sandboxes.json in every flip and autoflip path (state stays under the container name); status prints EST end, minutes left, ledger state and watcher; graduate-list checks state against data/darklaunch files and docs/knobs.md (READY/HOLD/PROBLEM, exit 1 on a problem); clean CLOSE writes graduation_note graduation-ready. 80 unit tests pass. Verified live read-only: autoflip plan read both sandboxes by alias, status and graduate-list ran on the real state. Not verified live: a CLOSE with the new note (the watcher runs the old code until the coordinator restarts it). Nothing build-specific."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

autoflip plan could not read the sandboxes by alias; status had no remaining time; graduate-list was unchecked. Done when the four fixes are in with tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 sandbox aliases (doofer, questy) resolve from tools/flip/sandboxes.json in every flip and autoflip subcommand
- [x] #2 flip.py status shows remaining minutes and the window end in EST plus the ledger state of each knob
- [x] #3 autoflip graduate-list checks each row against the per-knob ledger files, with unit tests
- [x] #4 a clean close writes graduation_note graduation-ready to the state and graduate-list prints it
- [ ] #5 live: autoflip plan reads both sandboxes after merge (watcher restart by the coordinator)
<!-- AC:END -->

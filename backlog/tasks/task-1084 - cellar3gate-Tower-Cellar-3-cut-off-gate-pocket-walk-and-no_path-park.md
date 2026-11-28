---
id: TASK-1084
title: 'cellar3gate: Tower Cellar 3 cut-off gate, pocket walk and no_path park'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-28T00:25:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 519000
card: "cellar3gate"
lane: next
owns:
  - "internal/nav/exit.go"
  - "internal/nav/cellar3gate_test.go"
  - "internal/questboard/"
  - "docs/probes/cellar3-gate-capture.md"
exclusive: []
contracts: []
result: "Synthetic Tower Cellar 3 fixture (two maps) reproduces the logged cut-off: the pocket click was refused by the mouth gate and dropped; pocketTarget now prefers a clickable cell, errors carry cause=cut_off|exit_trigger, the board parks no_path. Offline only; real grid capture is an open live checklist (U). Remaining (live): AC 3 capture the real Tower Cellar 3 grid (checklist in docs/probes/cellar3-gate-capture.md)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: Doofer parked the Countess chunk twice on a cut-off gate in Tower Cellar 3. Done when: the fixture passes and the park says no_path with the exit and cause.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The pocket walk avoids clicks the mouth gate refuses (two synthetic maps, must-fail pair)
- [x] #2 A chunk that fails on a no-path with a cause parks as no_path with the exit and cause
- [ ] #3 Real Tower Cellar 3 grid captured live (checklist in docs/probes/cellar3-gate-capture.md)
<!-- AC:END -->

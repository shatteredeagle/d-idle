---
id: TASK-1122
title: 'sweep1: sweep of the 13 fable reviews of 2026-11-28, untracked findings listed, offline-testable OPEN ones fixed'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T06:27:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 533000
card: "sweep1"
lane: next
owns:
  - "docs/probes/review-sweep-2026-11-28.md"
  - "internal/diffready/"
  - "internal/bot/diffready_test.go"
  - "internal/combat/move.go"
  - "internal/combat/whirl.go"
  - "internal/combat/fight.go"
  - "internal/combat/move_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Sweep the 13 review docs in docs/design/fable-review-2026-11-28, list what no card tracks, fix the offline-testable OPEN findings with must-fail tests. Done when the sweep doc is written and each fix has a test.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/probes/review-sweep-2026-11-28.md lists every finding of reviews 01 to 13 with severity and status
- [x] #2 review 12 C3: a read never deletes the difficulty go; another or unknown hero cannot use it (must-fail test, two classes)
- [x] #3 review 3 finding 5 (part): a gap landing asks the cave-mouth gate; gap casts and whirl legs count as fight casts (must-fail test, Barbarian and Paladin)
<!-- AC:END -->

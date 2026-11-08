---
id: TASK-133
title: 'skillunlocks: per-class skill unlock table, leveler rule and the one free-choice point'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:27:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies:
  - TASK-130
  - TASK-132
priority: medium
ordinal: 125000
card: "skillunlocks"
owns:
  - "internal/plandata/unlocks-sorceress.json"
  - "internal/leveler/unlocks.go"
  - "internal/leveler/unlocks_test.go"
  - "docs/skill-unlocks.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 item 6, fun-ideas.md sections 12 and 13 card 7. Skills unlock by quest or find, never by currency. Kill: the v3 soak order changes (any refused row in the buildv3 replay test). Done when: the criteria below hold.
Note 2026-11-07: this card owns the Sorceress unlock file; the other six classes are TASK-143 (classunlocks), which does not duplicate it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Per-class unlock table as data (internal/plandata/unlocks-<class>.json, events quest/waypoint/boss/item/clvl); the leveler Rule gains an unlocked(name) refusal from the per-hero progress state; measured runs load an all-met table; unknown class or skill is an error naming it; test against two classes or builds
- [ ] #2 The player may spend ONE free-choice point on a Codex-unlocked skill that the build table skips, shown with its consequence; the buildv3 replay test shows no refused row
- [ ] #3 Codex skill icons light up on the unlock; docs/skill-unlocks.md says built vs verified live and uses only levels checked by skillreqcheck; go vet and go test -p 1 of the touched packages; leakcheck 0
<!-- AC:END -->

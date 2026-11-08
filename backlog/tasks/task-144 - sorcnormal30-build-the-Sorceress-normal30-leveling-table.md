---
id: TASK-144
title: 'sorcnormal30: build the Sorceress normal30 leveling table'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:03:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies:
  - TASK-132
priority: medium
ordinal: 133000
card: "sorcnormal30"
owns:
  - "internal/plandata/build-sorceress_normal30.json"
  - "internal/plandata/sorcnormal30_test.go"
  - "docs/sorcnormal30.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Sorceress has only endgame90 and fallback files (class-early-game.md repo facts). Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A normal30 leveling table for the Sorceress (early skills Fire Bolt, Ice Bolt, Fire Ball or Glacial Spike per docs/research/class-early-game.md section 1, tagged EXP) as data; every class now has one
- [ ] #2 Her early game no longer loads the endgame90 or fallback-lightning files; test with two builds
- [ ] #3 go vet and go test -p 1 ./internal/plandata; docs/sorcnormal30.md says built vs verified live; leakcheck 0
<!-- AC:END -->

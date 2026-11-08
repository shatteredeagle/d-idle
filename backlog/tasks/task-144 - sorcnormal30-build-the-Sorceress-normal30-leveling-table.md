---
id: TASK-144
title: 'sorcnormal30: build the Sorceress normal30 leveling table'
status: Done
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
  - "docs/plan/data/build-sorceress_normal30.json"
  - "docs/plan/leveling.md"
  - "docs/wizardclass.md"
  - "cmd/d2dsetup/wizardclass.go"
  - "cmd/d2dsetup/wizardclass_test.go"
  - "cmd/d2dsetup/d2dsetup_test.go"
  - "cmd/d2dsetup/testdata/classes.golden.json"
  - "internal/plandata/plandata_test.go"
  - "internal/plandata/unlocks_class_test.go"
exclusive: []
contracts: []
result: "Built and tested offline, not verified live. Rows 2-30 equal v3 (Lightning line); #1 deviates from the fire-path wording by brief order, open choice in docs/sorcnormal30.md. Default and gate tables unchanged (opt-in sorceress_normal30). Owns widened to the wizard and test files."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Sorceress has only endgame90 and fallback files (class-early-game.md repo facts). Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A normal30 leveling table for the Sorceress (early skills Fire Bolt, Ice Bolt, Fire Ball or Glacial Spike per docs/research/class-early-game.md section 1, tagged EXP) as data; every class now has one
- [x] #2 Her early game no longer loads the endgame90 or fallback-lightning files; test with two builds
- [x] #3 go vet and go test -p 1 ./internal/plandata; docs/sorcnormal30.md says built vs verified live; leakcheck 0
<!-- AC:END -->

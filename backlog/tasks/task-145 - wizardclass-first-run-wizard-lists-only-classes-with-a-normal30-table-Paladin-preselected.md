---
id: TASK-145
title: >-
  wizardclass: first-run wizard lists only classes with a normal30 table,
  Paladin preselected
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:03:00Z'
updated_date: '2026-11-08T05:49:00Z'
labels:
  - container
  - feature
  - tester
  - onboarding
milestone: m-1
dependencies:
  - TASK-144
priority: medium
ordinal: 134000
card: "wizardclass"
owns:
  - "cmd/d2dsetup/wizardclass.go"
  - "cmd/d2dsetup/wizardclass_test.go"
  - "docs/wizardclass.md"
  - "cmd/d2dsetup/tables.go"
  - "cmd/d2dsetup/tables.json"
  - "cmd/d2dsetup/wizard.go"
  - "cmd/d2dsetup/d2dsetup_test.go"
  - "cmd/d2dsetup/testdata"
  - "cmd/d2dsetup/page/app.js"
  - "cmd/d2dsetup/README.md"
exclusive: []
contracts: []
result: "Built and tested offline, not verified live: class list from plandata (normal30; Sorceress via fallback table, TASK-144 open), Paladin preselected, unknown class error, golden, wizard.env D2D_CLASS 2 and 4; page syntax only. docs/wizardclass.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: user decision F11 (2026-11-07): the friend picks at the wizard, default Paladin. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The wizard's class list comes from data (the plandata build tables that have a normal30 table), not a literal; Paladin preselected; the Sorceress is listed now through the fallback table (interim rule, docs/wizardclass.md)
- [x] #2 An unknown class is an error naming it; test with two classes
- [x] #3 go vet and go test -p 1 ./cmd/d2dsetup; docs/wizardclass.md says built vs verified live; leakcheck 0
<!-- AC:END -->

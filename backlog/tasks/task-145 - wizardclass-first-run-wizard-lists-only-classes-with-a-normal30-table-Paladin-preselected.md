---
id: TASK-145
title: 'wizardclass: first-run wizard lists only classes with a normal30 table, Paladin preselected'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:03:00Z'
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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: user decision F11 (2026-11-07): the friend picks at the wizard, default Paladin. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The wizard's class list comes from data (the plandata build tables that have a normal30 table), not a literal; Paladin preselected; the Sorceress appears once TASK-144 lands
- [ ] #2 An unknown class is an error naming it; test with two classes
- [ ] #3 go vet and go test -p 1 ./cmd/d2dsetup; docs/wizardclass.md says built vs verified live; leakcheck 0
<!-- AC:END -->

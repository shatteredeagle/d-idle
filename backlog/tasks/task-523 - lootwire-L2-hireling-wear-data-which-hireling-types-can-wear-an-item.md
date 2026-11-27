---
id: TASK-523
title: 'lootwire-L2: hireling wear data (which hireling types can wear an item)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T02:10:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 283000
card: "lootwireL2"
owns:
  - "internal/merc/wear.go"
  - "internal/merc/wear.json"
  - "internal/merc/wear_test.go"
exclusive: []
contracts: []
result: "Hireling wear as data: merc.WearTypes/CanWear/TypeChain/WTypeGaps with wear.json (same rule as merc.Allowed, checked by test for 4 hirelings x 6 item kinds). Offline only; WTypeGaps not run against the real Hireling.txt."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: usable_by rules need to know which hirelings can wear an item. Done when: a data table answers it. Consumer: lootwire-L3.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 WearTypes(kind-tree of an item) answers which hireling types can wear it, from data
- [x] #2 tests for two hirelings and two item sets, unknown type errors name it
<!-- AC:END -->

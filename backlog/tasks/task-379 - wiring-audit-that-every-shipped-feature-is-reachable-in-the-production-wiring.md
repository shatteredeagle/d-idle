---
id: TASK-379
title: 'wiring: audit that every shipped feature is reachable in the production wiring'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-12T20:13:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 241000
card: "wiring"
owns:
  - "cmd/d2bot/wiring_test.go"
  - "cmd/d2bot/townforward.go"
  - "tools/wiringaudit"
  - "docs/wiring.md"
exclusive: []
contracts: []
result: "Fixed hidden-method gaps: fighterAdapter.SetPoverty, townAdapter ForgetBelt/PlannedSpend/UseWeapon/ReconcileMarks/MarkOf, mercTown hiding BeltFiller and the same parts, d2denv.Reader.ItemName; loot memory fixed by goldprio. cmd/d2bot/wiring_test.go fails when an adapter hides an optional method. Open gaps in docs/wiring.md (selfupdate IdleSafe, dead knob kiteLifePct, Hell rules). Verified offline and by sandbox log grep, not live-tested after the fix"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the loot memory never ran in production because looterAdapter hid its methods. Done when: docs/wiring.md matrix, wiring_test.go, fixes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Static audit of optional-interface assertions and nil-able hooks, orphan and unread-knob lists
- [x] #2 Production-wiring matrix test fails when an adapter hides an optional method
- [x] #3 Wiring gaps found are fixed with tests or listed with cause in docs/wiring.md
- [x] #4 Live-log grep separates no-opportunity from unwired
<!-- AC:END -->

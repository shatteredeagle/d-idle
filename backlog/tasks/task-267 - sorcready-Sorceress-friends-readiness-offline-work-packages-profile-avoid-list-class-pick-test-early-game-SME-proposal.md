---
id: TASK-267
title: >-
  sorcready: Sorceress friends-readiness offline work packages (profile avoid
  list, class-pick test, early-game SME proposal)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T18:46:00Z'
updated_date: '2026-11-10T19:55:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 196000
card: "sorcready"
owns:
  - "docs/sorceress-early.md"
  - "docs/smoke.md"
  - "internal/combat/classdata.go"
  - "internal/combat/classdata/profiles.json"
  - "internal/combat/sorcready_test.go"
  - "internal/combat/pure_test.go"
  - "internal/combat/castgap_test.go"
  - "cmd/d2bot/sorcready_test.go"
  - "cmd/d2bot/class.go"
  - "cmd/d2bot/class_test.go"
  - "internal/leveler/build.go"
  - "internal/leveler/buildv3_test.go"
  - "internal/leveler/palzeal_test.go"
  - "internal/leveler/unlocks_test.go"
  - "internal/leveler/early_test.go"
  - "internal/plandata/build-sorceress_normal30.json"
  - "internal/plandata/unlocks-sorceress.json"
  - "internal/plandata/sorcnormal30_test.go"
  - "docs/plan/data/build-sorceress_normal30.json"
  - "docs/plan/data/unlocks-sorceress.json"
  - "docs/sorcnormal30.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Follow-up sorcready2 applies the user decisions of 2026-11-10: fresh Sorceress defaults to sorceress_normal30; planForClass refuses another class's table (error names both classes); Warmth at clvl 2; cold line from clvl 12 (Ice Bolt, Ice Blast, Teleport 18, Glacial Spike, Blizzard 30) with a new sorceress/cold profile; Lightning and Chain Lightning avoided in lightning and cold profiles. Exit criterion: three fresh first-hour records before friends get her (docs/sorceress-early.md). Row order is an EXP d2-sme consult."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Wizard class pick vs normal30 table verified by test
- [x] #2 SME consult written as proposal, table unchanged, open questions listed
- [x] #3 Lightning profile avoids Chain Lightning and Lightning, tests on two classes
- [x] #4 Live measurement plan documented, not run
<!-- AC:END -->

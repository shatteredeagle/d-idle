---
id: TASK-699
title: 'countess: walk to her known spawn when the preset and hint fail on d2d'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-20T19:34:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 337000
card: "countess"
owns:
  - "internal/boss/act1.go"
  - "internal/boss/boss_test.go"
  - "internal/boss/script.go"
  - "internal/boss/run.go"
  - "tools/provingground/scenarios/act1.json"
  - "docs/probes/provingground-countess-2026-11-20.md"
exclusive: []
contracts: []
result: "Unit-tested only (go test -race -count=20 ./internal/boss). Live: NOT verified; act1.forgotten.countess was RED/BLOCKED on the builder (potion guard, record not observed), no determinism pair. level 25 levelmap lists no chest 580. See the probe doc. closecards3 Remaining (live): #2 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: on d2d the Countess has no monster preset, the Kolbot hint is refused by the nav mouth gate and explore finds nothing (docs/probes/provingground-countess-2026-11-20.md). Done when: Anchor.Spawns walks to her known spawn and the scenario is green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Unit test (Paladin and Sorceress arrival) fails without the fix, passes with it
- [ ] #2 act1.forgotten.countess GREEN Paladin 11 and Sorceress 3 on the builder (CHEATED setup, mechanics only)
<!-- AC:END -->

---
id: TASK-388
title: 'wiring2: fix the wiring gaps of the audit (updater guards, kite knob, Hell rules, dead hooks)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T23:34:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 245000
card: "wiring2"
owns:
  - "cmd/d2bot"
  - "internal/bot"
  - "internal/combat"
  - "internal/boss"
  - "internal/env"
  - "docs/wiring.md"
  - "docs/knobs.md"
  - "docs/stance.md"
  - "docs/aggression.md"
exclusive: []
contracts: []
result: "Updater IdleSafe/SelfTest/DiskFree set (nil skipped the checks); combat.kiteLifePct now drives the step-back line (default 60, stock unchanged); LevelGapOK, AntidotesToCarry and death weights called in Nightmare/Hell only (Normal unchanged, tested); boss OnPhase/OnKill removed, nav.Refill left on purpose. Open: belt fill never fires because mana potions fill the belt (docs/wiring.md). Offline tests only, not live-verified"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 selfupdate IdleSafe, SelfTest, DiskFree set in production
- [x] #2 combat.kiteLifePct read by the kite rule
- [x] #3 Hell rules called from planner, town visit and death handling, Normal unchanged
- [x] #4 dead boss hooks removed, nav.Refill decision documented
- [x] #5 docs/wiring.md matrix updated; production wiring test extended
<!-- AC:END -->

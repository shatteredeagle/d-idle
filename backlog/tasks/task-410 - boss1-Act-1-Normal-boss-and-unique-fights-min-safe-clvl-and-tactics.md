---
id: TASK-410
title: 'boss1: Act 1 Normal boss and unique fights, min safe clvl and tactics'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-14T11:29:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 251000
card: "boss1"
owns:
  - "docs/research/boss1-trace.md"
  - "tools/emu/bs_spawn.sh"
  - "tools/emu/bs_model.py"
exclusive: []
result: "docs/research/boss1-trace.md: Normal boss stats V run (game.exe room population, 4 seeds); the Smith is a superunique (404 life, x5 exp), Blood Raven 113, Andariel 1,024 (poison 80, fire -50); min safe clvl BR 8, Countess 10, Smith 16, Andariel 18-20 (I model); bot B1 class-blind kite, B2-B4 floors; no PR candidate; not verified live"
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the guided journey's Act 1 boss fights need a minimum safe clvl and a tactic per boss. Consumer: campaign chapter floors (data/campaign/normal.json) and the Andariel gate (internal/plandata/boss_levels.go). Done when: docs/research/boss1-trace.md carries the boss table (V run spawn stats), deviations and bot findings.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc with boss table, min safe clvl, deviations, bot findings, drill recipe
<!-- AC:END -->

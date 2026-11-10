---
id: TASK-218
title: 'ready-model: internal/readiness evaluator with checks as data rows'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
labels:
  - bot
  - feature
  - fidelity
dependencies:
  - TASK-214
priority: medium
ordinal: 167000
card: "ready-model"
owns:
  - "internal/readiness/"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. Evaluator, checks.json rows, /api/readiness, ResistGap into the risk meter. Resist basis is an explicit input (ResistRaw default, U) with tests for both cases; AC 3 stays open until TASK-216 settles it. Also touched outside owns, tiny: internal/bot (readinessfacts.go, runner.go, riskfacts.go), internal/watch/funwire.go (one field, one route), cmd/d2bot (readywire.go and 3 lines). Docs: docs/readiness.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). One pass/warn/fail check set per hero, target and difficulty, rolled up like the risk meter. Done when the evaluator and its table-driven tests pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/readiness evaluator, checks above as data rows per class and difficulty; unknown class/table is an error naming it; table-driven tests for Sorceress and Paladin; no new class literals in Go
- [x] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
- [ ] #3 Open question settled first: whether Player.Resist already includes the difficulty penalty (TODO in internal/combat/hard.go) is taken from the ready-audit result and the resist logic here uses that answer; a test covers both possible cases of the ambiguity until it is settled
<!-- AC:END -->

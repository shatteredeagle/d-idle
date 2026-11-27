---
id: TASK-742
title: 'pgselfcheck: hero self-check on the prep d2d and its static half in validate'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:48:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 348000
card: "pgselfcheck"
owns:
  - "internal/scenario/hero/selfcheck.go"
  - "internal/scenario/hero/selfcheck_test.go"
  - "cmd/d2scenario/validate.go"
  - "cmd/d2scenario/validate_test.go"
  - "cmd/d2scenario/cmds.go"
  - "cmd/d2scenario/run.go"
exclusive: []
contracts: []
lane: next
result: "Not verified live. The Proving Ground runner now reads the prepared hero on the prep d2d after setup (level, unspent stat and skill points, purse, potions) and ends the run BLOCKED as selfcheck NAMES on a mismatch; d2scenario validate runs the static half for every scenario and matrix class (reach goal met at the start with nothing else driving the bot, ground item before the position verb, unknown build, main attack not reached at the level with quest points 0; points scenarios are banked and skip the attack row; -gamedata adds the game learn rules). All 218 shipped scenarios pass. Life floor and the purse-below-kit literal are not done. Build-specific data stays in the plan tables."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design section 6.2, 7 card 8. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Self-check refuses a hero whose setup tags or state do not match the profile
- [x] #2 static half runs in validate offline
- [x] #3 must-fail test
<!-- AC:END -->

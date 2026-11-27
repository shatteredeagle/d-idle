---
id: TASK-1033
title: 'necroteeth: the Necromancer casts its main skill, not plain Attack'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T06:15:00Z'
updated_date: '2026-11-27T06:17:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 500000
card: "necroteeth"
lane: next
result: "Not verified live. Cause is the lab prep: the feature soak rows bank every stat and skill point (resource.pointsDecisions 0, no D2BOT_RUN), so no class skill is ever learned and every class kills with plain Attack; the bot data is fine (the default Necromancer plan learns Teeth at clvl 5 and binds it, test on Necromancer and Druid). Also dropped the Lower Resist on magic resistance curse rows from both Necromancer profiles (Lower Resist never lowers magic), must-fail test. Scenario format: pre.hero_profile (default played; banked for raw point writes and points.* scenarios), the prep spends the points through the hero library, tests on Sorceress and Necromancer. Build-specific data stays in the combat profiles and plan tables."
owns:
  - "internal/combat/classdata/profiles.json"
  - "internal/combat/necro_test.go"
  - "internal/leveler/mainattack_test.go"
  - "internal/scenario/scenario.go"
  - "internal/scenario/scenario_test.go"
  - "internal/scenario/setup.go"
  - "cmd/d2scenario/run.go"
  - "cmd/d2scenario/validate.go"
  - "cmd/d2scenario/validate_test.go"
  - "cmd/d2scenario/herospend_test.go"
  - "docs/process/proving-ground.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause found offline: lab prep or bot (row logs read)
- [x] #2 Main attack of the default plan is learned by clvl 12 when the bot spends the banked points (test on two builds)
- [x] #3 Necromancer profiles drop the Lower Resist when magicResist curse row (Lower Resist does not lower magic resist); must-fail test
- [x] #4 Prep fix for the featuresoak rows written in the report
- [ ] #5 Live: a clvl 12 Necromancer row with points spent casts Teeth
<!-- AC:END -->

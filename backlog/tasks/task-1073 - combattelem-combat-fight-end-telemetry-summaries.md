---
id: TASK-1073
title: 'combattelem: combat fight-end telemetry summaries'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T22:18:00Z'
updated_date: '2026-11-27T23:18:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 516000
card: "combattelem"
lane: next
owns:
  - "internal/combat/telem.go"
  - "internal/combat/telem_test.go"
  - "internal/combat/aim.go"
  - "internal/combat/fight.go"
  - "internal/combat/castfirst.go"
  - "internal/combat/stance.go"
  - "internal/scenario/skillid_test.go"
  - "tools/provingground/scenarios/combatdrills.json"
  - "docs/probes/combat-drills-2026-11-27.md"
exclusive: []
contracts: []
result: "Fight-end Info lines (aim, stance, minion) behind combat.deliveryUpkeep above 0, test that every pre.skills id exists for its class (all ids were already right), four scenarios assert the lines. Offline only, NOT run live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 aim, stance and minion fight-end summary lines
- [x] #2 pre.skills ids tested against the skills table
- [x] #3 scenario assertions use the summaries
<!-- AC:END -->

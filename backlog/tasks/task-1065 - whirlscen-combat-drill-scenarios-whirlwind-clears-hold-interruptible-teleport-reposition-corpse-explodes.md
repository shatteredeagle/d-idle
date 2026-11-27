---
id: TASK-1065
title: >-
  whirlscen: combat drill scenarios (whirlwind-clears, hold-interruptible,
  teleport-reposition, corpse-explodes)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T21:12:00Z'
updated_date: '2026-11-27T21:46:00Z'
labels:
  - bot
  - drill
dependencies: []
priority: medium
ordinal: 513000
card: "whirlscen"
lane: next
owns:
  - "tools/provingground/scenarios/combatdrills.json"
  - "tools/provingground/scenarios/tiers.json"
  - "internal/scenario/"
  - "docs/probes/combat-drills-2026-11-27.md"
result: "Offline only, NOT run live. Four scenarios plus the whirlwind control arm in combatdrills.json (T3 rows in tiers.json), new pre.skills setup verb (debug points), lab commands in docs/probes/combat-drills-2026-11-27.md. hold-interruptible patterns are U. Build-specific data lives in the scenario rows."
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TASK-734 AC1 and the night plan list name drills that do not exist. Done when: the scenarios validate and the lab commands are written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Scenarios authored and validated
- [x] #2 Lab run commands documented
<!-- AC:END -->

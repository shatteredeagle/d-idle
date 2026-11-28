---
id: TASK-1121
title: 'road4: d2shadow, offline digest of the shadow log lines (admission, planner, upkeep, loot) as dark-ledger evidence'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T06:27:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 533000
card: "road4"
lane: now
owns:
  - "internal/shadowlog/"
  - "cmd/d2shadow/"
  - "docs/shadowlog.md"
exclusive: []
contracts: []
result: "internal/shadowlog and cmd/d2shadow: one digest and verdict per shadow family (admission, planner, upkeep, loot) from recorded bot logs, -strict and -json; offline tests only (planner on Paladin and Necromancer, upkeep on a Sorceress and a Barbarian skill), not verified live and not yet run on a sandbox log. Nothing build-specific: classes and skills are read from the lines. Knob-free. docs/shadowlog.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: four dark-ledger rows (bot.admission, agency.planner, combat.deliveryUpkeep, loot.rulesEngine) name shadow log lines as their enable evidence and no tool reads those lines from a bot log. Done when: d2shadow LOG prints one verdict per family from a recorded bot log, offline.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 admission shadow lines digested by cell, purpose and case; verdict no-lines or review
- [x] #2 planner shadow lines digested per class (agree, attack and aura differences); tested on two classes
- [x] #3 upkeep shadow lines digested per skill and kind, first lines apart from repeats
- [x] #4 loot shadow lines digested per scope; ground verdict worse when today's pickup is lost
- [x] #5 cmd/d2shadow prints the digest (text or JSON), -strict exits 1 on a review verdict; must-fail tests
- [ ] #6 run d2shadow on one recorded sandbox night and cite the output in the ledger rows (live check)
<!-- AC:END -->

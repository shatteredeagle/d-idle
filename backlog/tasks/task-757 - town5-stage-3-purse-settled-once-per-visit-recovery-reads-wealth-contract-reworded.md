---
id: TASK-757
title: 'town5: stage 3 purse settled once per visit, recovery reads wealth, contract reworded'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-21T04:51:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 345000
card: "town5"
owns:
  - "internal/town"
  - "internal/resource"
  - "internal/bot/broke.go"
  - "internal/bot/leveling.go"
  - "tools/provingground/scenarios/townplan.json"
  - "docs/probes/town-plan-stage3-2026-11-22.md"
  - "docs/design/fable-review-2026-11-21/town-planner-b-model.md"
  - "town-basics"
exclusive: []
contracts:
  - "town-basics"
result: "Knob town.plan 3 (default stays 0): purse target per visit = carry line + plan spend, bank and shop-withdraw settle to it, recovery and poverty read purse plus stash. Contract sentence: 'At town.plan 3 or more the purse is settled once per visit: to the plan's spend of the visit plus the keep line, depositing the surplus and withdrawing the gap'. Offline: purse-target unit test passes, -race -count=20 on town, supply and bot recovery/poverty passes; the two-class rig tests (purserig_test.go) need the game tables: PASS on the builder host (first gate run caught my own wrong assertions, fixed), skip elsewhere. Scenarios NOT verified live: broke-after-bank Paladin GREEN already at stage 2 code, malus-errand-gold BLOCKED twice by the bot save point (purse settle seen in the bot log), Sorceress not run; setups CHEATED. Remaining (live): scenarios broke-after-bank and malus-errand-gold (AC 3). Status stays In Progress. Remaining (live): AC 3 scenarios broke-after-bank and malus-errand-gold (not in the night chain; add to the tail of Run 1)."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the bank leaves the purse at the carry line, so a hero with gold in the stash is judged broke (I9) and a chunk with a gold line is starved (I10). Done when: at knob town.plan >= 3 one stash stop settles the purse to carry line + planned spend; recovery and poverty read wealth; default stays 0.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 one stash stop settles the purse to planned spend + chunk gold + carry line (300+50*clvl); recovery/poverty read wealth, unreadable stash counts 0 and is logged
- [x] #2 contract line 'Bank before the shop' reworded to 'the purse is settled once per visit to planned spend + chunk gold + carry line' and the sentence shown in the result
- [ ] #3 scenarios broke-after-bank, malus-errand-gold
<!-- AC:END -->

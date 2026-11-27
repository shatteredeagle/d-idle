---
id: TASK-722
title: 'dlv-upkeep-state: upkeep driver, state-observed'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-21T04:42:00Z'
updated_date: '2026-11-27T18:49:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 345000
card: "dlv-upkeep-state"
owns:
  - "internal/combat/"
  - "internal/data/"
  - "internal/env/tuning_combat.go"
  - "docs/knobs.md"
  - "data/darklaunch.json"
exclusive: []
contracts: []
result: "First slice built offline, NOT verified live and not in a Proving Ground run. internal/data reads the Skills.txt columns aurastate and auratargetstate and resolves each skill state to its states.txt id and a curse flag (cache schema 7). internal/combat/upkeep.go is the state-observed upkeep driver behind knob combat.deliveryUpkeep (default 0, dark-launch row; named like combat.deliveryExpect, the key form the knob table uses for combat.delivery.upkeep): 0 the old buff pass, 1 shadow (old pass acts, each lasting disagreement logged once and counted), 2 the driver acts: state id from the table, trusted only after it was seen present (a wrong id falls back to the clock, no recast spam), a skill whose state is a curse is never cast as a self buff, inside a fight a non-vital slot casts at most once in four casts and only above a 10 percent mana reserve (profile block upkeep with vital, shareEvery, manaReservePct; defaults of Q7), and a pass after each clear. Tests on Paladin Holy Shield and Sorceress Frozen Armor plus a Necromancer curse case; must-fail checked by disabling the budget and the state read. Build-specific data: none in Go; state ids come from the game tables, the vital list from the profile. Literal count unchanged (no literal added; buffStates and the curse list stay for mode 0 until the cleanup card). Scenario combat.buffs-stay-up runs offline as TestBuffsStayUp (15 virtual minutes, Sorceress armor and Barbarian shout, uptime >= 90 percent and a cast bound; must-fail checked), not through the scenario runner. NOT done: differential rows for war cry and Holy Shield states (need the fork reads live), the skill-named aura knob. go test -race -count=20 data, env, combat green on the worker host; whole-repo vet and test, check.sh, contracts, backlog, leakcheck clean; knob budget +1 net (needs the owner word). Remaining (live): differential rows for war-cry and Holy Shield states (AC 3)."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Upkeep driver for slots with observe.by state/aura/clock; exclusive groups; in-fight share data defaults per archetype (1 cast in 4, mana reserve 10 percent, Q7); post-fight pass; deletes buffStates, curse skip list in buff pass, skill-named aura knob. Depends on dlv-rotation, dlv-expect.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Scenario combat.buffs-stay-up RED first, then green for Sorceress and Barbarian
- [x] #2 Knob combat.delivery.upkeep
- [ ] #3 Differential rows for war-cry and Holy Shield states
- [x] #4 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #5 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #6 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->

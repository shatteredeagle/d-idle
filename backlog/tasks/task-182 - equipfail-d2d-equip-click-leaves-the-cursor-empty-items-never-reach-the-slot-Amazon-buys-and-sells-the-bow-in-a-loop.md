---
id: TASK-182
title: 'equipfail: d2d equip click leaves the cursor empty, items never reach the slot; Amazon buys and sells the bow in a loop'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T11:07:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: high
ordinal: 151000
card: "equipfail"
owns:
  - "docs/soaks/equipfail.md"
  - "docs/probes/equipfail/"
  - "internal/leveler/score.go"
  - "internal/leveler/equip.go"
  - "internal/leveler/equip_sim_test.go"
  - "internal/town/ammo.go"
  - "internal/town/ammo_test.go"
  - "internal/town/sim/"
  - "docs/probes/test-characters.md"
exclusive: []
contracts: []
result: "Fixed, verified live (lockstep d2d, 5 wall min, writes after start 0): Amazon wears Cap, Short Bow and Arrows, Barbarian and Sorceress 0 refused clicks (before 28/46 and 6+9+5). Causes: unidentified magic items planned by the leveler; two-handed bow refused while Javelin and Buckler are both worn.  About 90 virtual minutes (5 wall) per class, one seed. docs/soaks/equipfail.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Evidence: docs/probes/classpacing/ (TASK-147): 'leveler: equip retry/failed', 'slot click refused', 'town chore failed ammo: item 15 did not reach slot rArm' in 7 of 8 runs (minutes/*.jsonl, logs on the worker host); the Amazon buys a Short Bow, cannot equip it and sells it again (shop/sell lines in bot-ama1). Done when: the criterion holds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fresh Amazon, Barbarian and Sorceress on lockstep d2d wear the bought armor and bow (no 'did not reach slot' or 'cursor never held item' WARN) in a 90 minute run, writes after start 0
<!-- AC:END -->

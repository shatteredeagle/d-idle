---
id: TASK-1117
title: 'fix8b: missing combat profile parks loudly, one mana need for kit, shop and belt'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:55:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 533000
card: "fix8b"
lane: now
result: "Offline, not verified live. Fighter: plan error and selection error are separate, a good selection clears the selection error; the runner parks a hero without a profile in town (status and ops log: no combat profile for <class>/<build>) and resumes when it resolves; row no-combat-profile is hard; the class-aware start is repeated after a missing or bad first read. Mana need: shop rule, town trip, belt columns and Mana Burn rules ask resource.ManaNeedFor like the kit gate; a mana_heavy build in Normal gets the need-2 default columns (hp,hp,mp,mp), so the old loot test rule 'Normal ignores the build' is replaced: two internal/loot tests rewritten with must-fail arms for the Sorceress Blizzard and the Hammerdin tables. Build-specific data: none new (table key mana_heavy, classtraits.json manaNeed). Open: the kit reads the configured table name only, the shop also the auto-selected one."
owns:
  - "internal/combat/"
  - "cmd/d2bot/"
  - "internal/bot/"
  - "internal/resource/"
  - "internal/town/"
  - "internal/loot/"
  - "docs/invariants.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fable review C (docs/design/fable-review-2026-11-28/12-fix-review-security-pg-data-pipeline.md) B1 and B2 on fix8 (TASK-1097). Done when the three offline criteria are ticked.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A hero without a combat profile is parked in town with the line 'no combat profile for <class>/<build>' (status and ops log), row no-combat-profile is hard; must-fail test, two classes
- [x] #2 The selection error clears on the next good selection and the class-aware start is repeated after a missing or bad first read; must-fail tests, two classes
- [x] #3 Kit gate, shop rule, town trip, belt columns and Mana Burn rules ask resource.ManaNeedFor; matrix test over every plan build of all 7 classes x 3 difficulties
- [ ] #4 Live check: a mana_heavy Normal hero (Hammerdin or War Cry Barbarian) fills its two mana columns and leaves town with a complete kit
<!-- AC:END -->

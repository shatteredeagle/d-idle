---
id: TASK-1034
title: 'arbloop: no chicken and re-enter loop when only some families are under arbiter control'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T06:16:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 500000
card: "arbloop"
lane: next
owns:
  - "internal/bot/goal.go"
  - "internal/bot/arbloop_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Featuresoak pass 1 (2026-11-26): with bot.arbiterControl 1 and bot.arbiterFamilies 3 a poor necromancer looped chicken, Save and Exit, re-enter 172 times in one virtual hour. Done when the offline test passes and the loop is gone.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The trip's leg back to the field never leaves town under the resume line; the hero heals first (offline test, Necromancer and Paladin, arbiter off, families 3 and 255)
- [ ] #2 featuresoak arbiter3 necromancer seed 201 rerun shows Save and Exit near the base count and no mode-no-exit hit (lab)
<!-- AC:END -->

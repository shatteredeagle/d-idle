---
id: TASK-1034
title: 'arbloop: no chicken and re-enter loop when only some families are under arbiter control'
status: In Progress
assignee:
  - live-agent
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
result: "Cause: a chicken outside the trip's own action (the survive hook fighting mid-leg under families 3) re-entered the game in town with the trip still on its leg back to the field, which walked out at 9 of 45 life and chickened again, 169 times. Fix: in town below the resume line that leg ends the trip and the low-life gate heals and holds; shared by the chain and the arbiter chores hook, no build data. Offline test on Necromancer and Paladin, arbiter off, families 3 and 255; not verified live (lab rerun box open). Remaining (lab): AC 2 featuresoak arbiter3 necromancer seed 201 rerun. Remaining (live): AC 2, featuresoak arbiter3 necromancer seed 201 rerun (Run 3 of the night plan plus one --classes necromancer line)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Featuresoak pass 1 (2026-11-26): with bot.arbiterControl 1 and bot.arbiterFamilies 3 a poor necromancer looped chicken, Save and Exit, re-enter 172 times in one virtual hour. Done when the offline test passes and the loop is gone.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The trip's leg back to the field never leaves town under the resume line; the hero heals first (offline test, Necromancer and Paladin, arbiter off, families 3 and 255)
- [ ] #2 featuresoak arbiter3 necromancer seed 201 rerun shows Save and Exit near the base count and no mode-no-exit hit (lab)
<!-- AC:END -->

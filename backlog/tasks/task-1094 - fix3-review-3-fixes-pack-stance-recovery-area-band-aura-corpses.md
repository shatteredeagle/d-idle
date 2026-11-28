---
id: TASK-1094
title: 'fix3: review 3 fixes (pack stance, recovery area, band, aura, corpses)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T03:41:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 524000
card: "fix3"
result: "Not verified live (unit tests only, fake clock). Five review 3 fixes: insidePack leaves under the life floor at any distance, holds once clear, honours the pin hold, and its rows only shadow at knob 2 until a row carries enabled (stance data in classdata/profiles.json, build-specific: hammerdin, hammerdin_zeal, poisonnova); the recovery no-potion rule now covers the no-portal and empty-pick paths; the survival band rule runs after judging and keeps the current and the last safe zone; the planner aura keeps the profile rows (Paladin data); the corpse reservation lapses after 4 s (U) and with the area. Open: the hammer-in-pack scenario shows only the shadow line until a row is enabled."
lane: next
owns:
  - "internal/combat/"
  - "internal/bot/"
  - "internal/survival/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fixes for docs/design/fable-review-2026-11-28/03-combat-nav-survival.md findings 1 to 4 and 6. Done when the must-fail tests pass and the whole-repo checks are clean.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 insidePack leaves under the row's life floor at any hostile distance, never walks in hurt, honours the pin hold; the three rows stay shadow at knob 2 until a row carries enabled
- [x] #2 recovery no-potion exclusion applies without a portal scroll and when the plan pick is empty; the first field is the floor
- [x] #3 survival band rule runs after judging and never drops the current zone or the last safe zone
- [x] #4 planner aura keeps the profile's low-life, resting, crowd and immunity rules
- [x] #5 corpse reservation lapses after corpseHold and with the area; id 0 reserves nothing
- [ ] #6 live: combat.hammer-in-pack drill on a build whose hammerdin row carries enabled true
<!-- AC:END -->

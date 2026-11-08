---
id: TASK-174
title: 'mercnomana: merc hire check follows the belt plan'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:49:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies:
  - TASK-164
priority: low
ordinal: 142000
card: "mercnomana"
owns:
  - "cmd/d2bot/mercwire.go"
  - "cmd/d2bot/mercwire_test.go"
  - "docs/belt-plan.md"
exclusive: []
contracts: []
result: "Offline: kit() and kitFull() follow the belt plan (TestKitFollowsBeltPlan, Sorceress and Paladin). Not verified live. Widened owns by docs/belt-plan.md (one line). U: build name is not known in mercwire, so a mana-heavy build table is not consulted when only column overrides are set."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The merc hire check in cmd/d2bot/mercwire.go still asks for healing potions, so a hero on an all-mana belt (belt plan TASK-164, 'all mana') never hires a mercenary. Make the hire kit check follow the belt plan. Done when the acceptance criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The hire kit check (kit() in cmd/d2bot/mercwire.go) reads the belt plan: a mana-only belt counts as a full kit for hiring
- [x] #2 Test with two builds: a mixed belt still needs healing potions, an all-mana belt hires
<!-- AC:END -->

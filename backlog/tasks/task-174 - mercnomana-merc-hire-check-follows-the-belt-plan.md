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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The merc hire check in cmd/d2bot/mercwire.go still asks for healing potions, so a hero on an all-mana belt (belt plan TASK-164, 'all mana') never hires a mercenary. Make the hire kit check follow the belt plan. Done when the acceptance criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The hire kit check (kit() in cmd/d2bot/mercwire.go) reads the belt plan: a mana-only belt counts as a full kit for hiring
- [ ] #2 Test with two builds: a mixed belt still needs healing potions, an all-mana belt hires
<!-- AC:END -->

---
id: TASK-254
title: "palprofile-data: Paladin attack order as profile data (Attack and Sacrifice, no Smite, Prayer rest-only)"
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T16:55:00Z'
updated_date: '2026-11-10T17:15:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 183000
card: "palprofile-data"
owns:
  - "internal/combat/classdata/profiles.json"
  - "internal/combat/class_test.go"
  - "internal/combat/packfight_test.go"
  - "internal/combat/palprobe_test.go"
  - "docs/palprofile.md"
  - "docs/sme/class-paladin.md"
  - "docs/research/class-early-game.md"
exclusive: []
contracts: []
result: "Offline only (tests, not verified live). Data in classdata/profiles.json; Prayer already rest-only, mana-potion rule not data; cause of back-offs UNSURE. docs/palprofile.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Paladin attack bands changed as data, Sorceress unchanged, tests on both
- [x] #2 Docs fixed; Prayer and mana-potion parts stated as not data-expressible
<!-- AC:END -->

---
id: TASK-1022
title: 'necroaudit: Necromancer audit and default table fix'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T19:15:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 493000
card: "necroaudit"
lane: now
owns:
  - "docs/probes/necro-readiness-2026-11-26.md"
  - "docs/epics/epic-class-necromancer.md"
  - "internal/plandata/build-necromancer_summoner_normal30.json"
  - "cmd/d2bot/class_test.go"
  - "internal/leveler/necrosummoner_test.go"
  - "docs/plan/data/build-necromancer_summoner_normal30.json"
exclusive: []
contracts: []
result: "Audit of the Necromancer written; the default summoner table now names its combat profile (the main attack Corpse Explosion had no key before); not verified live, lab drill rows not rerun in this branch (lab on the worker host was busy)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: owner priority 2026-11-26. Done when: audit doc and default table fix merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 audit doc written
- [x] #2 default summoner table names its combat profile
<!-- AC:END -->

---
id: TASK-184
title: 'cpnav: Assassin cannot leave Blood Moor to Cold Plains (nav: click would cross an unwanted exit trigger)'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T11:07:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: low
ordinal: 153000
card: "cpnav"
owns:
  - "docs/soaks/cpnav.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Evidence: bot-ass1 line 'leveling area unreachable ... nav: click {81 344}: nav: no path: the click would cross the trigger area of an unwanted exit' (TASK-147, docs/probes/classpacing). Done when: the criterion holds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fresh Assassin reaches Cold Plains from Blood Moor without 'leveling area unreachable'
<!-- AC:END -->

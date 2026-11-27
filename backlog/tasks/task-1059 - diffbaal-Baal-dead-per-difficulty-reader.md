---
id: TASK-1059
title: 'diffbaal: Baal-dead-per-difficulty reader'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T19:46:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 513000
card: "diffbaal"
lane: next
owns:
  - "internal/quest/baaldone.go"
  - "internal/quest/baaldone_test.go"
exclusive: []
result: "Built offline, not verified live: quest.BaalDead reads the eve_of_destruction word per difficulty (flag position U), table test and bad-difficulty must-fail."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design section 6 card 3; feeds diffready baal_dead_in_current.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Reader of the Eve of Destruction quest word per difficulty (flag position U) with table test and must-fail
<!-- AC:END -->

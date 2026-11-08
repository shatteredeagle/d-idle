---
id: TASK-137
title: 'ladderrepeat: the same ladder question reopens every minute after an answer'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:43:00Z'
labels:
  - bot
  - bug
  - tester
milestone: m-1
dependencies: []
priority: medium
ordinal: 113000
card: "ladderrepeat"
owns:
  - "docs/ladder-repeat.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by TASK-25 (docs/tester-live.md): in a 1 h realtime session the t0.area.toohard question opened 17 times and t0.retreat.drink 9 times, each re-asked about every 60 s after the answer go anyway (the hero keeps heading to the same area); a person answering would be asked over and over. Done when: cause found (rule not learned, key changes with the kit state, or the decision repeats), a fix or a documented reason, and a drill showing at most one open question per key per N minutes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cause of the repeat identified with log lines
- [ ] #2 fix or documented decision; test with two builds
<!-- AC:END -->

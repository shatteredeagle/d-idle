---
id: TASK-142
title: >-
  paladinsmoke: fun-loop slices live on a Paladin (pace, Codex firsts,
  Corpsefire at the Den)
status: Done
assignee:
  - live-agent
created_date: '2026-11-08T03:03:00Z'
updated_date: '2026-11-08T04:29:00Z'
labels:
  - bot
  - drill
  - ladder
  - tester
milestone: m-4
dependencies:
  - TASK-140
  - TASK-128
  - TASK-106
priority: medium
ordinal: 131000
card: "paladinsmoke"
owns:
  - "docs/paladinsmoke.md"
exclusive: []
contracts: []
result: "Verified live (lockstep 707 virtual min, container 20 real min, Paladin, writes after start 0 by construction): slices answer; the Paladin makes 0 kills in lockstep and 13 kills in 20 realtime min (cards TASK-175, 176, 177, 178); details docs/paladinsmoke.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user decisions 2026-11-07 (F10, F11): the Paladin is the default test class, so the fun loop is smoked on it first. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Return screen, tally strip, Codex (and the policy sheet if merged) run live on a Paladin in a hidden d2d slot: 30 to 60 minutes of lockstep and about 20 realtime minutes in the container
- [x] #2 Record pace (clvl per minute vs the Sorceress), Codex firsts fired, Corpsefire at the Den (goal reached? when?) and what is empty; writes after start: 0
- [x] #3 docs/paladinsmoke.md says built vs verified live; leakcheck 0
<!-- AC:END -->

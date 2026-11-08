---
id: TASK-142
title: 'paladinsmoke: fun-loop slices live on a Paladin (pace, Codex firsts, Corpsefire at the Den)'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T03:03:00Z'
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user decisions 2026-11-07 (F10, F11): the Paladin is the default test class, so the fun loop is smoked on it first. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Return screen, tally strip, Codex (and the policy sheet if merged) run live on a Paladin in a hidden d2d slot: 30 to 60 minutes of lockstep and about 20 realtime minutes in the container
- [ ] #2 Record pace (clvl per minute vs the Sorceress), Codex firsts fired, Corpsefire at the Den (goal reached? when?) and what is empty; writes after start: 0
- [ ] #3 docs/paladinsmoke.md says built vs verified live; leakcheck 0
<!-- AC:END -->

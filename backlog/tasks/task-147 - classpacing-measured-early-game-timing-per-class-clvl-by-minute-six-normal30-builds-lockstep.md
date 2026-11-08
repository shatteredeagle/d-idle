---
id: TASK-147
title: 'classpacing: measured early-game timing per class (clvl by minute, six normal30 builds, lockstep)'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T03:03:00Z'
labels:
  - bot
  - drill
  - soak
  - ladder
milestone: m-4
dependencies:
  - TASK-142
  - TASK-144
priority: low
ordinal: 136000
card: "classpacing"
owns:
  - "docs/soaks/classpacing.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: the veteran table is an estimate, not measured; the return screen's recommended-level table (TASK-131) needs per-class data. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 For each of the six normal30 builds (Amazon, Assassin, Barbarian, Druid, Necromancer, Paladin; plus the Sorceress once TASK-144 lands) a lockstep soak records clvl by minute for the first 2 hours; writes after start: 0
- [ ] #2 docs/soaks/classpacing.md compares measured pace with the veteran estimate of docs/research/class-early-game.md section 5 and gives the per-class minutes by zone as data for the recommended-level table
- [ ] #3 leakcheck 0
<!-- AC:END -->

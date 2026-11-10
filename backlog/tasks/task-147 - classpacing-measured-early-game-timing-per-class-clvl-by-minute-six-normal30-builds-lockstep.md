---
id: TASK-147
title: "classpacing: measured early-game timing per class (clvl by minute, six normal30 builds, lockstep)"
status: Done
assignee:
  - live-agent
created_date: '2026-11-08T03:03:00Z'
updated_date: '2026-11-10T16:54:00Z'
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
result: "Measured 90 virtual minutes (not 2 hours) for seven tables, one seed each, writes after start 0, verified live on lockstep d2d. AC1 partial, AC2 done (minutes-by-zone only as area ids in minutes/*.jsonl), AC3 see leakcheck. Pace is dominated by the recovery loop and equip failures (cards TASK-182/183/184), so not yet a clean per-class ranking. Status sync 2026-11-08 (stays open): AC1 open (two-hour per-class soaks not run; 90 virtual minutes measured)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: the veteran table is an estimate, not measured; the return screen's recommended-level table (TASK-131) needs per-class data. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 For each of the six normal30 builds (Amazon, Assassin, Barbarian, Druid, Necromancer, Paladin; plus the Sorceress once TASK-144 lands) a lockstep soak records clvl by minute for the first 2 hours; writes after start: 0
- [x] #2 docs/soaks/classpacing.md compares measured pace with the veteran estimate of docs/research/class-early-game.md section 5 and gives the per-class minutes by zone as data for the recommended-level table
- [x] #3 leakcheck 0
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

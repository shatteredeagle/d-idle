---
id: TASK-119
title: 'g2h-record: Gate 2 wave 3a (g2h) result'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-07T23:14:00Z'
labels:
  - ops
  - research
  - gate
milestone: m-0
dependencies: []
priority: medium
ordinal: 112000
card: "g2hrecord"
owns:
  - "docs/soaks/gate2-multiseed-20261107-1811-g2h.md"
  - "docs/research/g2h-wave3a.md"
exclusive: []
result: "wave 3a (g2h, seeds 11/13/17): 3 of 3 seeds PASS every criterion, indicative only (3 seeds, 1 holdout, wave 3b not run per D1); g2g NOKILL failures on 13 and 17 gone (seed 17 at 10 of 10); merc alive share 31-42% (was 4-16%) at 1.6-2.5x deaths; heal caveat bounded in docs/research/g2h-wave3a.md. Offline analysis of finished runs, not verified live beyond the soaks themselves."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Collect and judge Gate 2 wave 3a. Done when the record and docs/research/g2h-wave3a.md are merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Wave 3a (seeds 11, 13, 17, tag g2h) collected with the rebuilt d2dsoak
- [x] #2 Per-seed table vs g2g, merc numbers and three-fix proof checklist written
- [x] #3 Heal caveat (reload at full life) estimated and verdict stated (indicative, 3 seeds)
<!-- AC:END -->

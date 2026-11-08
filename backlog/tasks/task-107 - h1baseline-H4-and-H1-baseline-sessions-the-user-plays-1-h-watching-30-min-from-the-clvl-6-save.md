---
id: TASK-107
title: 'h1baseline: H4 and H1 baseline sessions (the user plays 1 h watching, 30 min from the clvl 6 save)'
status: Proposed
assignee:
  - user
created_date: '2026-11-07T21:56:00Z'
updated_date: '2026-11-08T03:49:00Z'
labels:
  - docs
  - research
  - ladder
milestone: m-4
dependencies:
  - TASK-106
priority: high
ordinal: 100000
card: "h1baseline"
owns:
  - "docs/probes/h1-baseline.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: direction review section 7 step 2 (H4 and H1 baseline). Done when: the numbers and sentences are in docs/probes/h1-baseline.md.

Amended 2026-11-07 (ADR 0018): the H1/H4 numbers are also logged by playsession2 (TASK-134) with the fun-loop cards; the qdensity remark in the criteria is moot (TASK-108 is Deferred).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One hour of watching: boredom time noted from the session record
- [ ] #2 30 minutes from the clvl 6 save: real question count per 10 minutes and one sentence on whether it felt good
- [ ] #3 If the count is already 3 per 10 minutes and it feels good, the qdensity card is marked unneeded
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
first-hour principle: watch mode (live video) carries the opening and the return screen carries the absences; H4 measures time-to-tab-close with video only
<!-- SECTION:NOTES:END -->

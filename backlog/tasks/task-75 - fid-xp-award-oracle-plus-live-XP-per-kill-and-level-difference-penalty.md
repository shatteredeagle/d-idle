---
id: TASK-75
title: 'fid-xp-award: oracle plus live: XP per kill and level-difference penalty'
status: Done
assignee:
  - live-agent
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - d2d
  - readonly
  - d2d-combat
  - d2d-monsters
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: medium
ordinal: 73000
card: "fid-xp-award"
owns:
  - "docs/probes/fid/fid-xp-award.md"
exclusive: []
contracts: []
result: "AC1 proven (0 of 219,790 lines, real rules::kill_exp vs FUN_0057e2f0, upstream 7557a54d, V run). AC2 live read not done. Doc docs/probes/fid/fid-xp-award.md. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: ST10. Method/kind: readonly. Live game (host time): yes, one short kill count on game.exe (about 20 min) after the oracle. Size: S. Fix: none expected (Den +60 % was composition); upstream fix 01337ba3 exists. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Re-run the xp-award grid (119,790 lines) against current main; expected 0 differ (tolerance: 0 differing...; Live read of XP before/after a Fallen, Zombie and Dark Hunter kill at clvl 4 to 6 on game.exe (read-only);... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Re-run the xp-award grid (119,790 lines) against current main; expected 0 differ (tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR)
- [ ] #2 Live read of XP before/after a Fallen, Zombie and Dark Hunter kill at clvl 4 to 6 on game.exe (read-only); expected 18/33/known values of docs/research/xp-award-trace.md exactly
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

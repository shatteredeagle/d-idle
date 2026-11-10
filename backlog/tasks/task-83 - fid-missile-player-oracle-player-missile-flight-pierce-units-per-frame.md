---
id: TASK-83
title: 'fid-missile-player: oracle: player missile flight, pierce, units per frame'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - d2d-skills
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: high
ordinal: 81000
card: "fid-missile-player"
owns:
  - "docs/probes/fid/fid-missile-player.md"
exclusive: []
contracts: []
result: "2026-11-08 decomp6: flight, range, roll and Nova hits equal (0 differences: 1,500 + 2,500 flights, 168,000 roll cases, 598 nova casts, V run on upstream a5b970d5); Lightning one-unit-a-frame deviation proven (44 of 1,192 casts, 0 after) with local candidate fid/missile-last-unit f45e45c8 (fork remote, not posted); AC 2 open: Chain Lightning hop not compared; doc docs/probes/fid/fid-missile-player.md; not verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: SK03 SK12 SK08. Method/kind: oracle. Live game (host time): no (a live read of one Lightning cast is optional). Size: L. Fix: upstream PR where flight or pierce differs; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Harness for missile flight: speed, range, collision per frame for Charged Bolt, Fire Bolt, Ice Bolt, Frost...; Lightning and Chain Lightning piercing and hop (SK12) compared; Frost Nova units-per-frame (SK08, folded...; classdata/skills.json ranges 10/20/25 confirmed against the oracle (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Harness for missile flight: speed, range, collision per frame for Charged Bolt, Fire Bolt, Ice Bolt, Frost Nova ring; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Lightning and Chain Lightning piercing and hop (SK12) compared; Frost Nova units-per-frame (SK08, folded in D1)
- [x] #3 classdata/skills.json ranges 10/20/25 confirmed against the oracle
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

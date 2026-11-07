---
id: TASK-83
title: 'fid-missile-player: oracle: player missile flight, pierce, units per frame'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
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
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: SK03 SK12 SK08. Method/kind: oracle. Live game (host time): no (a live read of one Lightning cast is optional). Size: L. Fix: upstream PR where flight or pierce differs; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Harness for missile flight: speed, range, collision per frame for Charged Bolt, Fire Bolt, Ice Bolt, Frost...; Lightning and Chain Lightning piercing and hop (SK12) compared; Frost Nova units-per-frame (SK08, folded...; classdata/skills.json ranges 10/20/25 confirmed against the oracle (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Harness for missile flight: speed, range, collision per frame for Charged Bolt, Fire Bolt, Ice Bolt, Frost Nova ring; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Lightning and Chain Lightning piercing and hop (SK12) compared; Frost Nova units-per-frame (SK08, folded in D1)
- [ ] #3 classdata/skills.json ranges 10/20/25 confirmed against the oracle
<!-- AC:END -->

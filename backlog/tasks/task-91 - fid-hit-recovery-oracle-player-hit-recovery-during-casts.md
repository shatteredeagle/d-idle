---
id: TASK-91
title: 'fid-hit-recovery: oracle: player hit recovery during casts'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:04:00Z'
updated_date: '2026-11-08T17:18:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - d2d-skills
  - d2d-time
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: medium
ordinal: 89000
card: "fid-hit-recovery"
owns:
  - "docs/probes/fid/fid-hit-recovery.md"
exclusive: []
contracts: []
result: "NO DEVIATION in the covered paths (2026-11-08, agent decomp5, upstream 1453f853): get-hit length 110,836 cases 0 differ, decision 73,694 frames 0 differ, drawn frames 1,747,580 ticks 0 differ (monsters); the three drafts are merged; FHR breakpoints table in docs/probes/fid/fid-hit-recovery.md. U: mode-transition gate for a hit during a cast, player drawn frames. Offline."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: SK16. Method/kind: oracle. Live game (host time): no. Size: M. Fix: upstream PR (get-hit rule, length, render drafts; not all merged); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: FHR table 0/5/9/14/20/30/42/60/86/142 compared; tolerance: 0 differing cases in the oracle run; a...; fight.hpp:715 2-frame wait rule compared with game.exe; remaining get-hit drafts listed with merge state (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 FHR table 0/5/9/14/20/30/42/60/86/142 compared; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [x] #2 fight.hpp:715 2-frame wait rule compared with game.exe; remaining get-hit drafts listed with merge state
<!-- AC:END -->

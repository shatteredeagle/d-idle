---
id: TASK-85
title: 'fid-boss-mods: oracle: unique, champion and boss modifiers (merge pr-mods)'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - upstream-pr
  - d2d-monsters
dependencies: []
priority: medium
ordinal: 83000
card: "fid-boss-mods"
owns:
  - "docs/probes/fid/fid-boss-mods.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: MA06. Method/kind: upstream-pr. Live game (host time): no. Size: S. Fix: upstream PR (pr-mods draft); boss life/def/exp base-level D1; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pr-mods evidence re-run on current main (157,618 cases); expected 0 differ after the fix; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 At most two untraced immunities (Corpsefire, Bishibosh, Rakanishu, Treehead, Blood Raven) traced and compared
<!-- AC:END -->

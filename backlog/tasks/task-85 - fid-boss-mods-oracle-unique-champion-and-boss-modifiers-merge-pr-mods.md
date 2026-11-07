---
id: TASK-85
title: 'fid-boss-mods: oracle: unique, champion and boss modifiers (merge pr-mods)'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-monsters
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: medium
ordinal: 83000
card: "fid-boss-mods"
owns:
  - "docs/probes/fid/fid-boss-mods.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: MA06. Method/kind: upstream-pr. Live game (host time): no. Size: S. Fix: upstream PR (pr-mods draft); boss life/def/exp base-level D1; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: pr-mods evidence re-run on current main (157,618 cases); expected 0 differ after the fix; tolerance: 0...; At most two untraced immunities (Corpsefire, Bishibosh, Rakanishu, Treehead, Blood Raven) traced and compared (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pr-mods evidence re-run on current main (157,618 cases); expected 0 differ after the fix; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 At most two untraced immunities (Corpsefire, Bishibosh, Rakanishu, Treehead, Blood Raven) traced and compared
<!-- AC:END -->

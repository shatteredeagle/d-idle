---
id: TASK-85
title: 'fid-boss-mods: oracle: unique, champion and boss modifiers (merge pr-mods)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-10T16:54:00Z'
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
result: "2026-11-08 decomp6: pr-mods merged and equal for c/u/s (184,064 boss_level cases, 0 differences except sub-point life representation); Cold Enchanted chill length missing on d2d (13,283 -> 2,039 cases) with local candidate fid/cold-enchanted-length (fork remote, not posted, not zero after: two-enchant units and 49 top-level cases remain); doc docs/probes/fid/fid-boss-mods.md; not verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: MA06. Method/kind: upstream-pr. Live game (host time): no. Size: S. Fix: upstream PR (pr-mods draft); boss life/def/exp base-level D1; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: pr-mods evidence re-run on current main (157,618 cases); expected 0 differ after the fix; tolerance: 0...; At most two untraced immunities (Corpsefire, Bishibosh, Rakanishu, Treehead, Blood Raven) traced and compared (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 pr-mods evidence re-run on current main (157,618 cases); expected 0 differ after the fix; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [x] #2 At most two untraced immunities (Corpsefire, Bishibosh, Rakanishu, Treehead, Blood Raven) traced and compared
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->

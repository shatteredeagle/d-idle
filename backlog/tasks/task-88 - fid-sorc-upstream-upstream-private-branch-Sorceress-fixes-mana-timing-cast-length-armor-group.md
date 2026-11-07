---
id: TASK-88
title: 'fid-sorc-upstream: upstream: private-branch Sorceress fixes (mana timing, cast length, armor group)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T18:04:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-combat
  - d2d-skills
  - d2d-time
  - fidelity
milestone: m-2
dependencies:
  - TASK-92
priority: medium
ordinal: 86000
card: "fid-sorc-upstream"
owns:
  - "docs/probes/fid/fid-sorc-upstream.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: SK05 SK06 SK07. Method/kind: upstream-pr. Live game (host time): no. Size: M. Fix: upstream PR from private branches (local/sorc-mana-timing-r2, shims R01, R03); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: lab-binary rebase. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: SK05: 42,168/42,168 differ on main; the private fix 0/53,628 differ; PR draft for it; tolerance: 0...; SK06: Lightning/Chain Lightning 19 vs 13 ticks PR draft; SK07: group stacking D2 PR draft; trigger D5 (any damage vs physical) traced (and 1 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SK05: 42,168/42,168 differ on main; the private fix 0/53,628 differ; PR draft for it; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 SK06: Lightning/Chain Lightning 19 vs 13 ticks PR draft
- [ ] #3 SK07: group stacking D2 PR draft; trigger D5 (any damage vs physical) traced
- [ ] #4 Docs/d2d-deviations.md shims R01 and R03 updated with the PR state
<!-- AC:END -->

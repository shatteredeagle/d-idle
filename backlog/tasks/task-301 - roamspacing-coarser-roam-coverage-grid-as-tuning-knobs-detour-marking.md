---
id: TASK-301
title: 'roamspacing: coarser roam coverage grid as tuning knobs, detour marking'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T14:55:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 209000
card: "roamspacing"
owns:
  - "internal/nav/roam.go"
  - "internal/nav/covview.go"
  - "internal/nav/roamstudy_test.go"
  - "docs/roamspacing.md"
  - "internal/nav/nav.go"
  - "internal/nav/stuckmem.go"
  - "internal/nav/roam_test.go"
  - "internal/env/tuning_nav.go"
  - "internal/bot/observe.go"
  - "cmd/d2bot/sensesfeed.go"
  - "docs/knobs.md"
  - "docs/nav.md"
  - "docs/roamleg.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Defaults 16/10 (simulator: max gap 14 subtiles, path -50 percent, legs -80 percent vs 7/5); NotePosition from Runner.trackAlive; see docs/roamspacing.md. Build-agnostic (no class data)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the 7/5 roam grid (1593 points in Blood Moor) is far finer than the 30-subtile clear radius needs, and ground passed on detours was never marked. Done when: knobs, new defaults on measured evidence, NotePosition marking.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 nav.roamSpacing and nav.roamCover knobs with clamp, view and payload follow them
- [x] #2 NotePosition marks coverage on every bot observation, overlay updates at once
- [x] #3 Spacing study table and new defaults in docs/roamspacing.md
<!-- AC:END -->

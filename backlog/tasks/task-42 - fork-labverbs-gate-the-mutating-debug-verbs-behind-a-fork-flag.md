---
id: TASK-42
title: 'fork-labverbs: gate the mutating debug verbs behind a fork flag'
status: Proposed
assignee: []
created_date: '2026-11-06T16:02:00Z'
labels:
  - tester
dependencies: []
priority: medium
ordinal: 42000
card: "fork-labverbs"
owns:
  - "docs/lab-verbs.md"
exclusive:
  - "fork-branch:d2d.lab3-labverbs"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/d2d-container/README.md: at the pinned fork ref the debug verb group answers unconditionally on the container-internal socket. Proposed: --lab-verbs (env D2D_LAB_VERBS=1), passed by the lab and gate harness, never by the tester entrypoint. Work is in the private fork, committed and pushed the same day.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fork: build and ctest pass with the flag off and on; read-only verbs (dump, level, players, warps, portals, presets, grid, nearby, itemat, missiles, states, blocked) stay ungated; commit is on the fork remote
- [ ] #2 Bot play path unchanged without the flag (a short bot run on the gated build); the doc lists every gated verb
- [ ] #3 Pin moved in the tester image; the bot-repo gate and drill launch flags updated (the private ops side is reported, not committed here); says verified live or not
<!-- AC:END -->

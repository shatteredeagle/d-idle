---
id: TASK-189
title: 'watchlayout: video inline at the top, contiguous minimap'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T15:22:00Z'
labels:
  - watch
  - feature
  - G-C
milestone: m-4
dependencies: []
priority: high
ordinal: 154000
card: "watchlayout"
owns:
  - "internal/watch"
  - "internal/senses"
  - "cmd/d2bot/ladder.go"
  - "docs/senses-map.md"
  - "docs/first-session.md"
exclusive: []
contracts: []
result: "Offline proof 2026-11-08: go test ./internal/watch ./internal/senses (WatchPage, OldVideo, AppJS, SensesJS via node, Stitch, adjacency tests) pass. Fixtures only, NOT verified in a browser or live: the Blood Moor grid is synthetic, only the recorded town image and its Adjacent rect are real; d2d alignment U (live check in docs/senses-map.md)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Video inline at the top of /watch with the minimap beside it
- [x] #2 Old video page links redirect to /watch
- [x] #3 Minimap stitches neighbour areas (Rogue Encampment to Blood Moor) with per-area fog
- [x] #4 Docs updated; live d2d alignment check listed
<!-- AC:END -->

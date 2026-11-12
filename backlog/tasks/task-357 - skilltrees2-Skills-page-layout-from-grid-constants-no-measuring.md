---
id: TASK-357
title: 'skilltrees2: Skills page layout from grid constants, no measuring'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T04:10:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 232000
card: "skilltrees2"
owns:
  - "internal/skilltree"
  - "internal/watch/web/skills.js"
  - "internal/watch/web/skills.css"
  - "internal/watch/skillsapi_test.go"
  - "internal/watch/testdata/skills_paladin.json"
  - "internal/watch/testdata/skills_sorceress.json"
  - "docs/skilltrees.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user feedback on /watch/skills: the first render was too large and off grid, lines ran off the tree until a later redraw (browser measuring before layout settled). Done when: layout is a pure function of server numbers.

result: geometry (x,y,w,h, links with edge-midpoint end points) now comes from internal/skilltree; the page scales board, tiles and lines as one unit with no measuring. Verified offline: unit tests, goldens, uilint, headless renders at 1400, 900 and 390 px for Paladin, Sorceress, Necromancer and Druid against a real install, early and late DOM identical. Not verified: hover in a real browser, live hero.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tile cells from skilldesc row/column; geometry and link end points computed server side
- [x] #2 Board scaled as one unit (aspect-ratio, percent tiles, same-viewBox SVG), no browser measuring
- [x] #3 Layout tests: unique cells, link ends on tile edge midpoints; headless early vs late identical
- [x] #4 Gates: package tests, uilint, contracts/backlog, leakcheck
<!-- AC:END -->

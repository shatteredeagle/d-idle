---
id: TASK-797
title: 'gridwhole: probe, is debug grid whole at arrival (maze levels included)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T02:18:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - d2d
  - probe
dependencies: []
priority: medium
ordinal: 367000
card: "gridwhole"
owns:
  - "docs/probes/gridwhole-2026-11-22.md"
  - "tools/probes/gridwhole.py"
exclusive: []
contracts: []
result: "Verified on the lab d2d (builder long lane, devp18 binary, CHEATED setup, mechanics only): debug grid 5 is whole at arrival on every one of 64 levels probed (106 runs, mazes included: Tower Cellar 1-5, Catacombs 1-4, Jail 1-3, Crypt, Mausoleum, caves, acts 2-5): exactly 5W x 5H cells, only . and #. Not literally 0 differences after all rooms are up: 0 in 75 of 106 runs, at most 103 compact cells (0.15 percent) otherwise, cause U (object footprints), never a region. The mouthgoal_test comment is wrong: Tower Cellar 5 is 150x150 subtiles, (21,70) and (121,70) are inside. Not verified: act 2-5 beyond seed 5, cell-by-cell residue. Build-specific: none."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: review docs/design/fable-review-2026-11-22/map-knowledge.md section 5 card 1. Done when the probe doc has the verdict.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Probe script tools/probes/gridwhole.py runs on the lab d2d
- [x] #2 Grid at arrival compared with the grid after every room is up and with the engine's collision query, maze levels included
- [x] #3 docs/probes/gridwhole-2026-11-22.md has the per-level table and settles the mouthgoal_test comment
<!-- AC:END -->

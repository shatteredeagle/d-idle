---
id: TASK-803
title: 'exitsread: the engine''s link table against the bot''s exit data; drop the Uber levels from the route graph'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T04:54:00Z'
labels:
  - d2d
  - probe
dependencies: []
priority: medium
ordinal: 368000
card: "exitsread"
owns:
  - "docs/probes/exitsread-2026-11-22.md"
  - "tools/probes/exitsread.py"
  - "tools/probes/exitscmp"
  - "internal/nav/hints_export.go"
  - "internal/nav/exitsread_test.go"
  - "internal/nav/route.go"
exclusive: []
contracts: []
result: "Verified on the lab d2d (builder, devp18 binary, CHEATED setup, link facts only): 40 levels (cloister chain, acts 1 outdoors, caves, towers, act 2-5 arrival levels) compared against Levels.txt Vis and the hint tables: every engine exit is an edge in the data; Monastery Gate to Outer Cloister and Outer Cloister to Barracks are border links, not warps, so the stuck step is not a data hole (border strip readability, cause U, not read here). One wrong data edge fixed: Burial Grounds to Pandemonium Run 1 (Uber level, portal only), must-fail test TestVisLinkToPortalOnlyLevelIsNotAnEdge. The strip read at the Monastery Gate was done later by card mapverify (docs/probes/strip-monastery-2026-11-22.md: the strip was never built from the layout verb, fixed in d2dworld placeLinked). Not done: acts 2-5 beyond arrival levels. Build-specific: none."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: review docs/design/fable-review-2026-11-22/map-knowledge.md card 4. Done when the mismatches are listed and the data ones fixed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Probe of the engine's debug warps and debug nearby per level, compared with Vis and the hint tables
- [x] #2 Every mismatch listed in docs/probes/exitsread-2026-11-22.md
- [x] #3 Data mismatch fixed with a must-fail test (Burial Grounds to Pandemonium Run 1)
- [x] #4 Border strip of the Monastery Gate to Outer Cloister read live (done by card mapverify: docs/probes/strip-monastery-2026-11-22.md, V on the lab d2d, CHEATED setup)
<!-- AC:END -->

## Why

Card navexit21 (keepBorders) did not fix Gate 1 seed 17: the re-run (ms2, bot 893e5ab3) shows 197 `nav: no exit to that level: level 1 from 2` in the first 90 seconds, although d2dworld reported "2 adjacent entries" on arrival in the Blood Moor. The border was never lost; the gate was dropped afterwards.

## What Changes

- nav `borderGates` looks up the cell across the border in the neighbour's grid in the neighbour's own frame (d2dworld's default OriginNone gives every level its own zero-based frame). Before, our coordinates were used as they were, so seed 17's Blood Moor gate cells were tested against the camp's wall column and all dropped.
- Recorded replies of a real d2d on seed 17 (level, layout, nearby, grids, probes of both levels) as fixture under internal/d2dworld/testdata/seed17, and a replay test in nav.
- docs/navexit22.md: evidence, tags, live checklist.

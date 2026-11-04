## Why

BACKLOG item 13 (user, 2026-10-31): the stream's map pane drew one level at a time, so crossing from the Rogue Encampment into Blood Moor, or from Blood Moor into Cold Plains, replaced the map. Diablo II's automap keeps one store per Levels.txt layer, so those levels are one map; caves and dungeons are not.

## What Changes

- cmd/d2dview: the map pane works in the act frame and draws a composite of every visited level of the player's layer group (rules and level list: docs/stream/map-contiguity.md).
- The pages swap the new map in when it has loaded and keep the trail across a stitched crossing.
- Delta: dashboard-pub ADDED requirements. No knob: always on.

## Impact

Owns docs/stream/map-contiguity.md and the cmd/d2dview files listed in tasks.md. Not live-verified: live check written in the doc.

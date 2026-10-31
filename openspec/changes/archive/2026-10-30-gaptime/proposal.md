## Why

GAPCASTS (agent/gapcasts 21f0dd4) found that cast commands are not casts on d2d and that the per-kill time gap (d2d 1.5 to 2.5x slower per kill) is not casting. Confirm with real-cast counting and split the time per kill into walk, cast and wait on both backends.

## What Changes

- cmd/d2dcasts2: d2dcasts plus castGapMs flag, virtual time per kill
- docs/d2d-casts-real.md: real casts per kill at castGapMs 450 and 650
- docs/research/per-kill-time-split.md: seconds per kill by component, density vs walking vs idle

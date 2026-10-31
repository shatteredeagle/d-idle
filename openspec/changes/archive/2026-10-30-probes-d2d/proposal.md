## Why

The SME-reviewed risk map (docs/research/d2d-fidelity-riskmap.md, branch agent/riskmap) ranks three d2d-side probes first; the user approved them on 2026-10-30: tick rate versus the 25 fps original, per-area spawn counts, and casts per kill. The original-game side is announced separately.

## What Changes

- docs/d2d-tick-rate.md: how d2d lockstep ticks map to 25 fps game time and virtual hours (code plus one measured check).
- docs/probes/d2d-spawns.csv and docs/d2d-spawns.md: monsters per area over 20 seeds, in a format the original side can fill.
- docs/d2d-casts.md: casts per kill vs Fallen in Blood Moor with a Charged Bolt Sorceress.
- cmd/d2dspawn (ticks, spawns) and cmd/d2dcasts, small attach-mode tools, plus docs/probes/run-*.sh wrappers.

## Why

KILLGAP (docs/killgap.md) found that the d2d bot needs 23 s per kill against 9 s on the original, 75 % of it walking, and that the 20 longest gaps are mostly empty-area roaming and a cave-mouth border shuttle. Remove the two top causes.

## What Changes

- Empty-area leave: a sweep with no kill for `Config.EmptyAreaSec` calls a non-quest area cleared (planner takes the next area or the fresh-game path). Off by default, d2d default in cmd/d2bot/d2dbackend.go.
- Cave mouth: `nav.mouthStep` falls back to a flood search when walls block the straight way away from the warp.
- Measurement on d2d, 4 seeds x 2 virtual hours, same load (parallel slot runs): docs/killfix.md.

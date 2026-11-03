## Why

game.exe 1.14d has no 10 s waypoint arrival cooldown in single player (docs/research/waypoint-trace.md); d2d dropped its own in upstream #153. The bot's wait is wasted time.

## What Changes

- `nav.wpCooldownSec` defaults to 0 (no wait), range 0..20, deprecated; positive values still wait. Retry round unchanged.
- Docs: docs/wpcooldown.md, status banners in docs/waypoints.md and docs/fixwp.md.

## Impact

internal/nav/waypoint.go, internal/env/tuning_nav.go, tests, docs. No spec delta (the knob is not in a spec).

## Overlap (declared)

docs/knobs.md is owned by fixcastgap; this card edits only the `nav.wpCooldownSec` row.

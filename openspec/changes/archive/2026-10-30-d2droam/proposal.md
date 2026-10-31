## Why

On d2d the world source returns whole level maps, so Navigator.Explore reports ErrExplored at once and sweepPass marks an area cleared about 3 s after arrival with 0 kills (docs/d2d-pace.md section 'level'). The campaign never kills, earns no xp or gold.

## What Changes

- a FullyKnown capability on the d2dworld world source; nav roams coverage points (spacing about the aggro radius, nearest-first tour, unreachable points skipped, leg cap) instead of exploring unknown cells
- Game.exe exploration unchanged
- bot logs `sweep: coverage N/M points, kills K, cleared=<bool>`

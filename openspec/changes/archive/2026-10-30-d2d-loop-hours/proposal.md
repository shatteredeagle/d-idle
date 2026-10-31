## Why

ADR 0009 lists hours of the full bot loop on d2d as unproven (U). Run the full bot loop on d2d for several hours in lockstep and record stability, progression and pacing, so the default flip and the replicated soak rest on data. Evidence goes to docs/d2d-loop-hours.md.

## What Changes

- run the full loop for at least 4 virtual hours per seed, 3 seeds
- record crashes, stalls, loop gaps (loot, merc, Act 2 are known stubs)
- report gaps as follow-up cards

## Why

Same-seed full-bot runs on lab6 stay identical only to tick ~2400, then the bot itself sends a different command (`debug grid 5`). Cause found: the world's grid-refetch rate limit read the wall clock. Fix it, mask the remaining start-up noise, and re-run the pair test and the full-bot replay.

## What Changes

- d2dworld refetch timer on the lockstep clock (d2denv, d2dcombat wiring)
- journal raw reply side file (D2DCTL_JOURNAL_RAW) to find differing fields
- pair.sh: lab6, slot pool, PAR runs at once
- docs/d2d-determinism.md section 6

## Why

On d2d the bot needs 23.6 s per kill vs 9.1 s on the original (GAPTIME, docs/research/per-kill-time-split.md); the gap is idle and wandering between kills, not casting. Instrument per-kill timing, find the top causes on hidden d2d, fix them.

## What Changes

- `killtime` log line per kill (walk/cast/wait buckets), aggregated by `d2dsoak extract`
- measurement on d2d, causes in docs/killgap.md
- fixes for the top 1-2 causes

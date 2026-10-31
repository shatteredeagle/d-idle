## Why

The d2d bot spends about 75 % of its per-kill time walking (docs/research/per-kill-time-split.md) and its walking speed against the original was never measured; hit recovery (riskmap rank 8) is only covered for the cast branch of `00623f50` (PR #128). Trace both in game.exe 1.14d (offline, the unicorn oracle) and compare with d2d numerically.

## What Changes

- docs/research/movement-and-recovery-trace.md: walk/run speed, stamina, the hit-recovery decision and length, with addresses and a quantity / original / d2d / ratio / evidence table
- tools/emu/move_recovery.py: the oracle that runs game.exe's own hit-size test, get-hit rate and path step under unicorn
- docs/research/upstream-pr-candidates-movement.md: a draft PR body if d2d has a real bug (nothing posted)

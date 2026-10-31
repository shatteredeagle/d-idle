## Why

Two fixes were never measured live: the fresh-game livelock fix (livelockfix 5f864c2) and the run policy (runfix 9923e7c). This change integrates both in a scratch branch and measures them: one livelock check, then a 4 seed x 2 virtual hour A/B (D2D_RUN=off vs on, interleaved, same binary).

## What Changes

- tools/runab/: runner script for one slot run.
- docs/d2d-run-ab.md: the measurements.

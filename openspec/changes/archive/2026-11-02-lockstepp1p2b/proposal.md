## Why

Continuation of lockstepp1p2 (preparation, merged). The coordinator started a hidden instance, so probe P1 (read the 25 fps frame counter) and P2 (proxy hook, k=1 control then k=2) can run. User go 2026-11-02 for the hook set "d2-timehook proxy ijl11 IAT clock scaling" (instrumentation, ADR 0014).

## What Changes

- cmd/lockstepscan: read-only counter scanner and watcher (no writes).
- docs/research/lockstep-p1p2.md: P1 numbers, P2 state, what failed.

## Impact

Docs and a read-only tool. No spec, policy, ADR or glossary change. Overlap (declared): continues the archived card lockstepp1p2, which owned the same doc.

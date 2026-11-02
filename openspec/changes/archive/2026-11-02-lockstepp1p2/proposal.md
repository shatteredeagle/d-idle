## Why

docs/research/original-game-lockstep.md section 4 defines probes P1 (read the game frame counter, confirm the 25 fps model) and P2 (proxy DLL, null hook k=1 control, then k=2). The user gave a go on 2026-11-02 for the hook set "d2-timehook proxy ijl11 IAT clock scaling" as INSTRUMENTATION (ADR 0014).

## What Changes

- docs/research/lockstep-p1p2.md: evidence, tables, what failed, what is blocked.
- The hook source lives in a separate private repository and is not part of this repo.

## Impact

Docs only. No spec, policy, ADR or glossary change. No game was driven in this stage (see the doc).

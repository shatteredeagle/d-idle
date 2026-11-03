## Why

The first-pass missiles coverage trace (card covmissiles) ran game.exe against the table and the real d2d rules and added two ledger lines (228 rows). A first pass can hold a re-derived number or a harness blind spot, so an independent second reading and run is due before the lines are trusted.

## What Changes

- docs/research/verifycovmissiles-trace.md: verdict per claim (VERIFIED with counts, HOLDS WITH LIMITS, NOT VERIFIED, WRONG), my own reading of `FUN_005adf10` and its hit / damage tables, a new deviation (pierce: game.exe rolls up to four pierces at creation from stats 166 and 156 seeded by pierce_idx, d2d reads stat 328 as a per-hit percent), bot finding checks, drill recipe, matrix text.
- tools/emu/vcm_*: own harnesses (flight with perturbed records, the d2d loader dump, the hit roll with my own cases, the pierce creation function, disassembly helpers).

## Impact

Docs and emulator scripts only. No Go change, no live game, no d2d slot, nothing pushed. The ledger is not changed (the author's lines are not on main yet; the trace says which edits to make once they are). No spec, policy, ADR or glossary change.

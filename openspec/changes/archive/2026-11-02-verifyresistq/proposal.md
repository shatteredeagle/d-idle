## Why

The first pass of VERIFY-SKRESIST (verify-skresist.md) re-checked Q1, Q2, Q4, Q5, Q6, Q8 and Q10 and left Q3 (boss / champion / unique modifiers), Q7 (curse values) and Q9 (what an immune hit still does) at U. This second pass checks those three with its own harness.

## What Changes

- `docs/research/verifyresistq.md`: verdicts for Q3, Q7, Q9 with case counts and seeds, differences from the author's trace, bot findings, limits.
- `tools/emu/vq_dis.py`, `vq_mods.py`, `vq_gethit.py`, `vq_curse.py`: the harness (game.exe's FUN_005a1370 and FUN_0057cb00 under the emulator on the worker host).
- No Go code, no game, nothing pushed; no ledger row changed (no d2d side was run).

## Overlap (declared)

Same topic as the card verifyskresist (different questions, different doc); no shared files.

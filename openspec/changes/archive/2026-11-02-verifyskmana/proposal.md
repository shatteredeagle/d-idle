## Why

Independent re-check of docs/research/sk-mana-trace.md (queue item VERIFY-SKMANA): the mana check and payment claims need a second look with a separate harness.

## What Changes

Adds docs/research/verify-skmana.md and tools/emu/vsm_diff.py, vsm_dis.py. Verdict: the check, the payment with the minmana floor, the silent refusal, the zero-cost return and the shapeshift exemption hold; the do-ignores-refusal claim is not tested here.

## Impact

No spec delta, docs and emulator scripts only. Overlap (declared): none.

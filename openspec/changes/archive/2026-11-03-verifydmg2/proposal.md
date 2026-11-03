## Why

Second independent verification of DMGROLL round 2 (private fork branch `local/batch2-monsterdmg-r2`, head b9ecb415): are the first verifier's two findings fixed, are the new claims (Energy Shield in 256ths, life under one point is 0) right, is the PR an improvement on every compared quantity, and is it ready for a PR.

## What Changes

- `docs/research/verify-dmgroll2.md`: the verdict, per claim VERIFIED / HOLDS WITH LIMITS / NOT VERIFIED / WRONG.
- `docs/research/verify-dmgroll2-run.txt`: the verifier's own differential log.
- `tools/emu/v2_*` and `tools/emu/vd_xref.py`: the verifier's harness (game.exe's FUN_0057b420 run natively, a game-order model of FUN_0057c1e0 from the read, d2d's own `monster_blow` / `absorb_blows` of base and r2), read tools, CI and suites script.
- Nothing pushed, nothing posted; the fork and the author's tree and build were only read.

## Impact

Docs and tools/emu only. Overlap (declared): reads the dmgroll2 card's draft and tools; `tools/emu/vd_xref.py` is the first verifier's file (verifydmg, not merged to main) copied unchanged.

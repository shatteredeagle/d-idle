## Why

Fourth and final independent verification of DMGROLL round 4 (private fork branch `local/batch2-monsterdmg-r4`, head 9ea2afe9): is every kept part (the roll, `reduce_fine`, `exact`, the Bone Armor pool in 256ths off the unreduced field, the guard, the sub-point life rule, the pet and merc carry) at 0 differences against game.exe through the REAL compiled `Fight::absorb`, is r4 never worse than base with the Energy Shield parts removed, is the ponytail note accurate, and is it ready for a PR.

## What Changes

- `docs/research/verify-dmgroll4.md`: the verdict, per claim VERIFIED / HOLDS WITH LIMITS / NOT VERIFIED / WRONG.
- `docs/research/verify-dmgroll4-run.txt`: the verifier's own differential log (354,000 frames, seed 161803).
- `tools/emu/v4_*`: the verifier's harness: a probe linked against the r4 build's `libgame.a` that calls the real `Fight::absorb` and `Fight::hurt_side`, header glue for base and r4, the runner (game.exe's FUN_0057b420 natively plus the game-order model of FUN_0057c1e0), the git export and the CI and suites script.
- Nothing pushed, nothing posted; the fork and the author's tree and build were only read.

## Impact

Docs and tools/emu only. Overlap (declared): reads the dmgroll card's draft and this verifier's round 3 tools (`tools/emu/w3_*`, adapted as `v4_*`).

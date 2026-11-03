## Why

Third independent verification of DMGROLL round 3 (private fork branch `local/batch2-monsterdmg-r3`, head 2df7d1eb): is the author's correction of the second report right (fixed records still reach FUN_0057c1e0), is the order absorb, then flat, then percent right for Bone Armor and Energy Shield, is r3 never worse than base on any compared quantity, are the missing unit tests acceptable, and is it ready for a PR.

## What Changes

- `docs/research/verify-dmgroll3.md`: the verdict, per claim VERIFIED / HOLDS WITH LIMITS / NOT VERIFIED / WRONG.
- `docs/research/verify-dmgroll3-run.txt`: the verifier's own differential log.
- `tools/emu/w3_*`: the verifier's harness (game.exe's FUN_0057b420 run natively, a game-order model of FUN_0057c1e0 with Bone Armor and Energy Shield from the reads, d2d's own `monster_blow` / `absorb_blows` / `shield_blows` of base, r2 and r3 with each tree's `Fight::absorb` glue), the disassembly helper, the git export and the CI and suites script.
- Nothing pushed, nothing posted; the fork and the author's tree and build were only read.

## Impact

Docs and tools/emu only. Overlap (declared): reads the dmgroll card's draft and the second verifier's tools (`tools/emu/v2_*`, copied as `w3_*` and changed).

## Why

The SEEDDIST run (docs/research/seeddist-trace.md, D3) quantified one real deviation: game.exe rolls a monster's physical damage as `FUN_0057b420` (min << 8 + rand((max - min) << 8), top excluded, 256ths); d2d's `monster_blow` rolls whole points inclusive. The user decided to fix it if it is still open on upstream main, privately: no push to the public fork, no PR, no upstream comment.

## What Changes

- On the private d2d fork (branch `local/batch2-monsterdmg`, one commit on upstream main `3e073888`): `monster_blow` rolls the physical in 256ths as the game does and carries the part of a point (`Taken::frac`, `Foe::frac`, `StruckBlow::frac`) to the player's life, to pets and to the merc; tests next to the neighbours in `tests/test_combat.cpp`.
- In this repo: `tools/emu/mdmg*` (the differential: game.exe's function under the emulator against the real `monster_blow` of any tree), `docs/research/dmgroll-draft.md` (step 1 answers, the differential, parity, limits, commit text).
- Nothing pushed to the public fork, no PR.

## Impact

Docs and tools/emu in this repo. Overlap (declared): reads docs/research/seeddist-trace.md, monster-attack-trace.md and pr-candidates-20261102.md; no other card writes `monster_blow`.

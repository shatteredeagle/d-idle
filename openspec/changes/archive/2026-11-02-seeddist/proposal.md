## Why

Seed-for-seed equality between d2d and game.exe is out of reach (docs/research/seed-study.md): game.exe gives every unit its own seed, a step of the game seed with high word 666, while d2d rolls monster life on one world stream and drops on a unit seed of its own game-seed model. The A:B plan (docs/research/tierb-design-20261102.md) is statistical, so the open question is whether the outcome DISTRIBUTIONS of the two backends agree for the Act 1 classes the early game meets, not whether single rolls do.

## What Changes

- On the worker host (not in this repo): a d2d-side sampler built by hand against a scratch upstream build, and a game-side sampler running game.exe 1.14d's own functions under the emulator (tools/emu/sd_*), for Fallen, Zombie, Skeleton, Quill Rat, Rogue (Dark Hunter), Gargantuan Beast, Blood Raven and Andariel: life roll, hit roll vs defence, damage roll, first drop, at least 20,000 monsters per class per side, plus a seed-for-seed control and a cross-check on game.exe's real room population.
- In this repo: docs/research/seeddist-trace.md (questions with addresses and tags, results with chi-square and Kolmogorov p-values, deviations D1.., bot findings B1.., drill recipe, matrix-cell text), tools/emu/sd_* scripts, one ledger row (formulas F16) and the regenerated docs/coverage/rule-coverage.md, merge-lines.txt for the shared files.
- Nothing pushed or posted; no PR candidate (no deviation that a fix could remove); no Go change.

## Impact

Docs and tools/emu only. Supports the statistical A:B (Tier B stays held) and lowers the RNG-stream blocker to "seed-for-seed lockstep only". Overlap (declared): none beyond reading docs/research/monster-stats-trace.md and treasure-trace.md.

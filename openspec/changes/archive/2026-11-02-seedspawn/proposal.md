# Proposal: seedspawn

## Why
seed-study.md (section 8, card 3) asks whether d2d's world at first load (layout, preset units, spawn list) equals game.exe's for the same map seed, and where the RNG streams differ after level generation. The decision it feeds: are d2d's outcomes statistically equivalent to game.exe's even though they are not seed-for-seed.

## What Changes
Docs and emulator scripts only: `docs/research/seedspawn-trace.md`, `tools/emu/ss_run.sh` (own prefix `ss_`), ledger rows only for what the run proves. Differential: game.exe's own code in the emulator against the built d2d `drlg-dump`, levels 1, 2, 3, 4, 8, at least 20 seeds each.

## Not in scope
No game start, no live run, no Go change, no d2d slot, no change to d2d or the bot.

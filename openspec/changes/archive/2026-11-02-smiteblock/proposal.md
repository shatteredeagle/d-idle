## Why

Two traces of Paladin Smite disagree: our docs/research/palskill-trace.md (D6) says a block stops Smite, the community trace docs/research/pal-skill-trace.md (D8) says Smite forces the hit after the block check. One emulator question settles it, and d2d's behaviour has to be counted against the answer.

## What Changes

- docs/research/smiteblock-trace.md: the Smite / block path in game.exe 1.14d (FUN_005ce9f0, FUN_0057ec10, FUN_0057dfb0, FUN_0057dbf0, FUN_0057d4f0), a native run of the chain (96,000 cases), d2d's `player_blow` run on the same question (80,000 blow pairs), deviations, bot findings, drill recipe, proposed matrix text.
- tools/emu/sb_*: the harness (`sb_block.py`, `sb_run.sh`, `sb_dbg.py`, `sb_d2d_test.cpp`, `sb_build.sh`, `sb_after.sh`).
- docs/coverage/ledger.tsv: note text of the existing `skills.pal 96,97,106` line (no new claim); docs/coverage/rule-coverage.md regenerated.

## Impact

Docs and emulator scripts only. No Go change, no spec, ADR, glossary or policy change. No PR candidate (the harness is new).

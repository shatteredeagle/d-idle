## Why

LEVELSPAWN found that d2d reads Level(N) / Level(H) for every Nightmare and Hell monster while game.exe takes the area's MonLvl (Levels.txt MonLvl2Ex / MonLvl3Ex), except noRatio and boss rows. This card builds the real-build differential on the private fork, writes the fix as a local candidate branch (`local/nmlevel`, two commits on upstream main 6935cf64), and drafts the upstream PR text. Nothing is opened or posted.

## What Changes

- docs/research/pr-nmlevel-draft.md: the PR draft, counts, WARN answers and what a narrow verifier must check.
- tools/emu/nm_pop.py: the real-population comparison (game.exe's census against d2d's `test_boss_level`). It runs the census script `tools/emu/ls_area.py` of card levelspawn (branch agent/levelspawn, not merged when this was written; the copy used is byte-identical to it).

## Impact

Docs and emulator tools only in this repo; the code change lives on the private fork branch. No spec, ADR or glossary change.

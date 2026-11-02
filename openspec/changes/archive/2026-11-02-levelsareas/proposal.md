# Proposal: levelsareas

## Why
Coverage gap: the Levels.txt rows (137) were untested. First pass compares game.exe area monster level (FUN_0061dca0, run natively) with the columns d2d holds and censuses the table.

## What Changes
Trace docs/research/levelsareas-trace.md; scripts tools/emu/lv_tbl.py and lv_area.py; a ledger row for the area-level aspect. No Go code, no live game.

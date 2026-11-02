## Why

Board item 16 (SKCURSEVAL, decomp-queue; source SKRESIST section 10): the curse values (Lower Resist `-dm56`, Conviction `-min(ln34,150)`, Amplify Damage, Decrepify, Weaken, Defense Curse) come from skill calcs. SKRESIST proved the 1/5 rule on immunes but did not compare the value calcs themselves at every level 1..99 with the synergy terms and the cap.

## What Changes

- Run game.exe's calc machine (FUN_00646ca0) under the emulator on every calc cell of the 11 curse rows at levels 1..99 against d2d's `eval_calc` (existing d2d-side test mode); count cases and mismatches, two seeds.
- Read the curse do (FUN_005c37c0, FUN_005c35c0, FUN_005c3400, FUN_005c37a0) for the level the calcs get and the length divisor.
- Trace doc `docs/research/skcurseval-trace.md` with Q, D, B, limits, drill recipe, matrix-cell text; tools `tools/emu/sk_curseval.py`, `sk_curseval_run.sh`, `sk_curseval_tbl.py`.
- Ledger: one line (rows 309, 310) and the regenerated `rule-coverage.md`. No Go change, no game, no d2d slot, nothing pushed.

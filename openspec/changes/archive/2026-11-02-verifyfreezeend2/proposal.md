## Why

Board item 11 VERIFY-AIDEL, second pass: the first pass (docs/research/verify-aidel.md) did not check the freeze-end claim and a think that comes due during a get-hit.

## What Changes

- docs/research/verifyfreezeend2-trace.md: Q1..Q6, deviations D1 (think due in a get-hit: game.exe runs the driver, d2d holds it) and D2, bot findings (none), drill recipe, matrix text.
- tools/emu: own harness (vfe2_run.py, vfe2_cmp.py, vfe2_tab.py, vfe2_all.sh, vfe2_build.sh, test_vfe2_gh.cpp). No Go code, no game, no d2d slot.

## Impact

Docs and emulator scripts only. No spec delta.

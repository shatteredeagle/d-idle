# verifycursedef: independent verification of SKCURSEVAL, SKDEFENDERS, SKSHIELD

## Why

The three first-pass decomposition docs (docs/research/skcurseval-trace.md, skdefenders-trace.md, skshield-trace.md) were
written by a Sonnet agent on credits; SKCURSEVAL was suspiciously cheap. An Opus verifier rebuilds the game.exe oracle from
scratch and checks each claim and the ledger rows before they are relied on.

## What Changes

- docs/research/verify-cursedef.md: per-claim verdicts (VERIFIED / HOLDS WITH LIMITS / NOT VERIFIED / WRONG) and ledger checks.
- tools/emu/vcd_*: the verifier's own game.exe calc oracle and d2d eval_calc probe (emulator on the worker host only).

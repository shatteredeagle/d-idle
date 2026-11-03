## Why

Coverage gap, first pass for `skills.pal`: 15 of the 30 Paladin rows (Charge, Vengeance, Conversion, Fist of the Heavens, Holy Shield, Redemption, Fanaticism and the resist / aim / meditation auras) had no ledger line, so `docs/coverage/rule-coverage.md` showed them UNTESTED.

## What Changes

- Run game.exe 1.14d's calc machine under the emulator against the real d2d rules for these rows (calc slots, to-hit, aura slots, element damage; seeds 11 and 29), count cases per row.
- Read the Redemption callback and the area search, and the Vengeance start function (Q1, Q2 of the trace); compare with d2d's source; bot findings by file and line.
- Trace: `docs/research/covpal-trace.md`; tools `tools/emu/covpal*.{py,sh}`.
- Ledger lines for the proved rows only, gap 6 removed from `gaps.tsv`, `rule-coverage.md` regenerated.
- No Go change, no live game, no d2d slot, no PR candidate.

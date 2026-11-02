# VERIFYCOVSTATES: independent verification of the states first pass

Docs and emulator scripts only; no Go change, no live game, no d2d slot.

## Why

`docs/research/covstates-trace.md` (card covstates) claims stun equals game.exe, a poison deviation of 12 of 2,520 cases, the state-bit layout and two ledger rows. An independent reading and run (own harness, own seeds, real MonStats rows on both sides) says which claims hold before the ledger rows are merged.

## What Changes

- `docs/research/verifycovstates-trace.md`: verdicts per claim, own counts, deviations D1.. (poison deviation is larger and two-directional), bot findings confirmed, limits.
- `tools/emu/vcs_*`: own game.exe / d2d harness, state-bit probe, disassembly helper.
- `docs/coverage/ledger.tsv`: rows states 21 and 2 with my counts; `docs/coverage/rule-coverage.md` regenerated.

## Overlap (declared)

`docs/coverage/ledger.tsv` and `docs/coverage/rule-coverage.md` are shared coverage files that every coverage card appends to; this card adds only two `states` lines (ids 21 and 2), which the covstates card also adds with its own counts: whichever merges second drops its duplicate.

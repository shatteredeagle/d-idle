## Why

The rule-coverage table (docs/coverage/rule-coverage.md) showed the `missiles` subsystem at 9 percent run-tested (684 rows of Missiles.txt). Missiles decide most of the damage the bot deals (Sorceress and Paladin bolts) and takes (monster missiles). First pass: run game.exe's own code on the table-driven missile columns and on the hit roll of the player skills not yet run, compare with d2d's real loader and rules, list deviations and bot findings.

## What Changes

- docs/research/covmissiles-trace.md: the trace (Q1..Q8, deviations D1..D4, bot findings B1..B3, no new PR candidate, drill recipe, matrix text, dead ends).
- tools/emu/cm_*: the differentials (hit roll of 8 more skills, every Missiles.txt row made by game.exe against the table and the real d2d loader / rules, helper scripts).
- docs/coverage/ledger.tsv and the regenerated docs/coverage/rule-coverage.md: 6 hit rows and 222 table / flight rows proven (missiles 9 percent to 42 percent run-tested).

## Impact

Docs and emulator scripts only. No Go change, no live game, no d2d slot, no d2d change (nothing pushed). No spec, policy, ADR or glossary change. Shared-file lines are in merge-lines.txt.

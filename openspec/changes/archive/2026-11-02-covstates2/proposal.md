# covstates2

## Why

The states table is 185 rows and 3 percent of it was run-tested. The first pass covered freeze, chill, armors and two curse rows. This second pass takes the monster-facing and common combat states that were still untested: curses (Amplify Damage, Weaken, Iron Maiden, Life Tap, Decrepify, Terror, Dim Vision, Confuse, Attract), Taunt, Battle Cry, Inner Sight, Slow Missiles, and the state-group rule behind the shape-shift forms.

## What Changes

- docs/research/covstates2-trace.md: how game.exe 1.14d puts a timed state (FUN_0056e970) and removes a state group (FUN_0056c740), against d2d's real put_state, deviations D1.., bot findings B1.., drill recipe, matrix-cell text.
- tools/emu/cs2_*: the harnesses (game.exe natively under the emulator; d2d's put_state compiled and called; the end-to-end script).
- docs/coverage/ledger.tsv rows for the states proven, and the regenerated docs/coverage/rule-coverage.md.

## Impact

Docs, emulator scripts and ledger rows only. No Go code, spec, ADR or glossary change; no game, no d2d slot; nothing pushed or posted.

## Why

The rule-coverage table (docs/coverage/rule-coverage.md) had `monster.ai` at 14 percent run-tested (121 of 146 AI types untested). The Act 2 to 5 monsters and the quest bosses (Duriel, the Summoner, Mephisto, Izual, Diablo) decide how long the bot's boss fights last and what it dodges. Second coverage pass: run game.exe's own think for the AI types not in the ledger and compare with d2d's real `rules::mon_think`, list deviations and bot findings.

## What Changes

- docs/research/covmonai2-trace.md: the trace (Q1..Q4, deviations D1..D5, bot findings B1..B2, no PR candidate, drill recipe, matrix text, dead ends and harness artifacts).
- docs/research/covmonai2-runs.txt: the raw run output (mismatch counts and acts per AI).
- tools/emu/mai3_*: the differential wrappers (scenes and label fixes, run, classifier, debug dump, table dump, ledger and pack helpers).
- docs/coverage/ledger.tsv and the regenerated docs/coverage/rule-coverage.md: 78 AIs TESTED-0-DIFF, 3 TESTED-DEVIATION (monster.ai untested 121 to 40 of 146).

## Impact

Docs and emulator scripts only. No Go change, no live game, no d2d slot, no d2d change (nothing pushed). No spec, policy, ADR or glossary change. Shared-file lines are in merge-lines.txt.

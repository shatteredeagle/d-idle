## Why

The Gate 2 criterion "hire the merc" (openspec/specs/soak-gates, openspec/specs/merc-policy, docs/merc-wire.md) must be confirmed by the memory reader, and docs/merc-wire.md says d2d exposes no hire-window reader. The question: what can the bot already observe about a hired merc on d2d, what is missing to confirm "hired, alive and following" and "revived", what is the smallest sound change, and how is it proven against game.exe.

## What Changes

- docs/research/hirereader-trace.md: the decomposition (Q1..Q5, deviation candidates D1..D4, bot findings B1..B5, drill recipe, proposed matrix text, dead ends).
- docs/research/hirereader-design.md: feasibility, the smallest d2d change (proposal only), the bot-side fixes, the proof plan and the order of work.
- tools/emu/hrd_pay.py, hrd_d2d_test.cpp, hrd_build.sh, hrd_run.sh: a new differential of game.exe's purse payment (FUN_00576d90) and revive chain (FUN_006637b0 + FUN_00576d90) against d2d's rules::hire and rules::resurrect_merc, 200,000 cases, 0 differ.
- No Go change, no d2d or fork change, no ledger row (the rule-coverage table has no merc economy subsystem yet; the proposed denominator and ledger lines are text in the trace).

## Impact

Docs and emulator tooling only. No spec, policy, ADR or glossary change. Nothing run live; nothing pushed. Missing glossary term for the coordinator: "purse" (carried gold first, then the stash) is used as in the d2d sources.

## Overlap (declared)

The doc docs/merc-wire.md and the bot files named in the bot findings belong to other cards; this card changes none of them and only describes the needed edits.

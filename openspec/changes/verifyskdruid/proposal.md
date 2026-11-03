# verifyskdruid

## Why

SKDRUID (docs/research/skdruid-trace.md) claims the 30 Druid skill rows' calc cells equal d2d (0 of 27,800) and gives summon caps and per-skill values. Before the ledger row it added (`skills.dru` 30 / 30 TESTED-0-DIFF) is trusted, an independent verifier re-checks the claims with its own reading, its own harness and its own seeds.

## What Changes

- docs/research/verifyskdruid.md: own reading of FUN_00646ca0 / FUN_00646460 / FUN_00646c00 / FUN_006442a0, own runs (seeds 101, 202: 30,680 cases, DIFF 0; negative control DIFF 78), claim-by-claim verdicts, bot findings, ledger assessment, drill recipe.
- tools/emu/vsd_* (own harness: blvl and lvl fed separately, stat-request log, calc offsets, read-back).

## Impact

Docs and emulator scripts only. No Go code, spec, ADR or glossary change; no game, no d2d slot; nothing pushed or posted. Proposed ledger split is text in the doc, not applied.

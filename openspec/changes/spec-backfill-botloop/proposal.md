## Why

The spec-sme review (2026-10-31) found that the archived cards navstall, botsmall and smallfixes changed observable bot-loop behaviour (the minute line restarts counter, the per-game reset, monster kind) while carrying skip_specs, and that no capability spec covers the bot loop. A reader of openspec/specs cannot learn these contracts.

## What Changes

- New capability `bot-loop` (delta ADDED, applied to openspec/specs/bot-loop/spec.md at merge): restarts counting, per-game memory forgetting through one helper, monster classification by flag bits.
- Specs only; no code is touched. Evidence stays in docs/bot-small.md.

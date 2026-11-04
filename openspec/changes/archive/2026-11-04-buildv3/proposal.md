## Why

The SKPASS replay showed the leveler spends a point on the next legal row, so the fresh Sorceress table ran in a different order than its rows say. The user approved the SME's rows on 2026-11-01.

## What Changes

- New table `build-fallback-lightning-v3.json` (opt-in by name; v2 stays the default until a live soak), a Chain Lightning attack band gated by the learned skill, the table's own `combat` key picks the combat line, tests, docs/buildv3.md.

## Impact

Data plus a small profile and selector change. No spec names the skill order (early-game-policy and soak-gates checked), so no delta spec. No ADR or glossary change.

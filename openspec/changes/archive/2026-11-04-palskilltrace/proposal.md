## Why

Work item PALSKILL (Issue #3): trace the melee dispatch of Smite, Zeal, and Sacrifice in game.exe 1.14d vs d2d's skill_built.
no observable behaviour change: docs and offline formula tooling only, no code under internal/** or cmd/**.

## What Changes

- Dedicated trace in `docs/research/pal-smite-zeal-sacrifice-trace.md`
- Cross-reference link in `docs/research/pal-skill-trace.md`
- Formula self-check helper in `tools/emu/verify_palskills.py` (arithmetic self-check of written formulas, I)
- No Go code modified

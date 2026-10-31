## Why

Our death gates (deaths per 1000 kills, fidelity-gate spec) assume d2d's monsters hurt the player as game.exe 1.14d's do. Soaky died to Fallen, Carvers, Rakanishu and a lightning-enchanted Carver in Stony Field; the riskmap ranks monster melee (9) and ranged (6) as unmeasured. This card traces the monster-to-player damage path in game.exe offline (disassembly, unicorn oracle) and compares it with d2d's rules.

## What Changes

- `docs/research/monster-attack-trace.md`: the rules (addresses, numbers, V/S/U), the d2d comparison per item, a death-risk model for a clvl 1-10 Sorceress against the Act 1 packs, open points.
- `tools/emu/monster_hit.py`: oracle script (runs from upstream's tools/emu) running FUN_0057d9b0 (hit chance), FUN_0057b420 (physical roll) and FUN_0057cb00 (get-hit) in game.exe's own code against ports of d2d's rules.
- `docs/research/upstream-pr-candidates-monster.md`: the one deviation found (get-hit rule), its proof status, why no PR branch was drafted.

owns: docs/research/monster-attack-trace.md, docs/research/upstream-pr-candidates-monster.md, tools/emu/monster_*.py, openspec/changes/decompmon/**
exclusive: none (offline; no game instance, no Go changes)
gates: none (research)

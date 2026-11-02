## Why

Decomp queue item 3 (PALSKILL): how game.exe 1.14d dispatches Smite, Zeal and Sacrifice (hit count, AR and damage bonus, mana and life cost, knockback and stun, what a held button repeats), which of them d2d builds, and where d2d and our bot deviate. The recon (docs/research/paladin-act1-recon.md section 6 rows 1 and 5) had them as unverified, with d2d believed to fall back to a plain attack.

## What Changes

- `docs/research/palskill-trace.md`: behaviour per skill (V read of FUN_005ce790, FUN_005ce8e0, FUN_005ce9f0, FUN_005daf40, FUN_005dbc60), deviations D1 to D9 with tags, bot findings B1 to B5, PR candidate status (none by the rule; before / after measurement for D1), drill recipe, proposed register text.
- Own harness `tools/emu/pal2*` (game.exe's calc machine, FUN_0057b420 and FUN_0057d9b0 under the emulator; d2d's real `eval_calc`, `skill_tohit`, `make_fighter`, `player_blow` in a probe build), seeds 7 and 1.
- `docs/coverage/ledger.tsv` one line (skills.pal 96, 97, 106) and the regenerated `docs/coverage/rule-coverage.md`.
- No Go code, no game, no d2d slot; nothing pushed or posted.

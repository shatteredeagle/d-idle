## Why

The verifier of the melee-blow PR candidate found that game.exe's damage build (`FUN_0057b7d0`) runs the undead / demon block for a player OR an attacker whose `FUN_006259b0` returns 2, so the candidate's `not_player` exclusion of the merc and the Shadow may state the wrong rule. This card finds what `FUN_006259b0` returns for each unit kind.

## What Changes

- Trace in game.exe 1.14d the getter (`FUN_006259b0`), the setter (`FUN_005543b0`, 21 call sites) and the spawn map of MonStats Align, with V read / V run / D / I tags.
- `tools/emu/al2_run.py` (native runs: Align byte of 734 rows, setter/getter round trips, getter shapes, the gate of `FUN_0057b7d0`) and two reading aids.
- `docs/research/alignment2-trace.md`: table unit kind -> alignment -> undead block, the rule d2d should implement, bot findings, drill recipe. No ledger row (the runs are not against d2d).
- No Go code, no live game, no PR candidate.

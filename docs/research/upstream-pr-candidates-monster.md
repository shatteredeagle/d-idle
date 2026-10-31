# Upstream PR candidates from the monster attack trace (agent DECOMPMON, 2026-10-30)

> status: research, current. Companion to `monster-attack-trace.md`. Nothing was pushed, posted, commented or filed on GitHub; no PR branch was made. USER RULE (2026-10-30): a PR needs a differential test against game.exe's own code (boundary cases and >= 100k random cases, mismatches shown before the patch and zero after, exact commands), proof that nothing else changes (upstream tests, unchanged state hashes), one claim, and no U link in the causal chain; otherwise report a finding. Standard: docs/process/upstream-pr-standard.md (agent PRPROOF; not landed when this was written).

## Candidate 1: a unit's get-hit test as FUN_0057cb00 (tiered by hit class), not a fixed fraction

- Claim (one): the flinch (GH) decision after a hit lands. game.exe `0057cb00(unit, record, hit class)`: no get-hit under 1 hp of damage; divisor by hit class (1..12: 16, 8, 16, 32, 64, 8, 16, 32, 16, 8, 8, 16; else 16); none under max life / div; under max life / (div / 2) one draw in 2 says none; under max life / (div / 4) one draw in 4 says none; a monster with its MonStats flag (`0046c140(row, 3)`) never. Called for the player at `0057d37a` and for a monster at `0057d0c8` (both in `0057cee0`, the "takes damage" event).
- d2d today: player `fight.cpp:4903` `damage x 12 >= max life` (always); monster `ai.cpp hurt` `damage x 8 >= max_hp`; both marked `ponytail` in upstream combat.md ("Monster hit recovery: at an eighth of max life. Player: a twelfth").
- Proof we have: `tools/emu/monster_hit.py gh` runs game.exe's `0057cb00` under unicorn with the max-life getter, the state test and the MonStats flag hooked, the seed draw native: 672 boundary cases (each tier edge, one above and below, for every hit class 0..13 and four max lives) + 100,000 random cases, 0 differ from the tiered port; 18,502 of the random cases differ from d2d's deterministic twelfth (the before-patch mismatch set). Commands in `monster-attack-trace.md` section 5.
- PROOF: incomplete. Missing: (a) the after-patch run against d2d's own binary (the port lives in this script, not in `rules::`; the rule must be moved into d2d and its test must call game.exe's function with the same hooks, like upstream's `kill_exp.py`); (b) upstream's test suite green on the patched tree, and state hashes of scripted scenarios unchanged except the GH field (`tests/`, `--scripted` runs: not run here, d2d not built in this card); (c) the record field `+0x28` (compared with the total at `0057cb25`: which damage kind it holds is U) and the hit-class source for a missile hit on the player (record `+0x60`, D says Missiles.txt HitClass; d2d uses `missile.info->hit_class`, S); (d) the state-1 and state-0x15 (stun) paths were read, not run; (e) the player's FHR tables (how long GH lasts) are out of this claim and stay `ponytail`.
- Expected effect on our metrics: second order (trace doc section 4 item 1); it changes when a cast is interrupted, in both directions by monster.
- Why no branch: (a) and (b) need a d2d build and run, outside this card's time box, and the rule would be a `U` link without them.

## Rejected (traced, no deviation)

- Hit chance, physical roll, damage application (256ths, dead under 1.0), the running-player rule, block a third while running: oracle 100k + 100k, 0 differ (section 1 and 2 of the trace doc).
- Lightning Enchanted bolts: same count, level, trigger cadence and damage (S + D); the sub-1-hp trigger corner is not worth a PR without a scenario that reaches it.
- "Monster level = area level at spawn" in the running d2d: not a deviation found, an open probe (item 2).

## Lead that needs a trace first

- The to-hit bonus the plain monster attack passes to `0057ec10` (wrapper `005a5470`): expected 0 (Skills.txt `Attack` has no ToHit), d2d's ai.cpp:412 adds the skill's ToHit + LevToHit x level for skill swings. One read of `005a5470`'s callers settles it.

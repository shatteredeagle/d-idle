# Why the original needs fewer Charged Bolt casts per kill than d2d (analysis)

> status: current. Agent GAPCASTS, 2026-10-30, offline. Inputs: `docs/probes/orig-casts.csv` and `docs/probes/d2d-casts.csv` (branch agent/origprobes 90b09fb), Skills.txt / MonStats.txt read with upstream `mpq-cat` from the 1.14d MPQs on the worker host, upstream d2decomped `e36922e` read-only. Tags: V = computed or read by me this run, S = secondary, U = unverified. Classic 1.14d. No game was started.

## Headline

**The gap is mostly a probe artifact: a "cast command" is not a cast on d2d.** The d2d csv has `mana_spent` per kill row. Charged Bolt costs (Skills.txt mana 24, lvlmana 4, manashift 5: (24 + 4 x (lvl-1)) x 32 / 256, V) 3.0 / 4.5 / 6.5 at skill 1 / 4 / 8, and the csv spends exactly that (3 at CB1; 4 or 5 at CB4; 6 or 7 at CB8, V). But rows with 2 commands spend one cast's mana in 65 of 73 rows at CB1 (all monsters), rows with 4 commands spend 2 casts (6), and 24 % of one-command rows spend 0 mana (V, csv). So about 1.6 to 1.8 commands (1.83 / 1.61 / 1.58) go into each real cast: a command issued while the previous cast is still playing is dropped by d2d.

Real d2d casts per Fallen kill (Fallen rows only: sum of mana_spent / cost / kills; mana is attributed to the row of the kill, so a fight's casts are split across rows of the same fight): see the table.

## Prediction table (casts per Fallen kill)

| CB level | bolts per cast | dmg per bolt (lightning, avg) | orig commands/kill (V-orig) | d2d commands/kill (V-d2d) | d2d casts/kill from mana (V, csv) | commands per cast on d2d |
|---|---|---|---|---|---|---|
| 1 | 3 | 4-8 (6) | 0.96 (1.12 per xp-kill) | 2.59 | 1.42 | 1.83 |
| 4 | 6 | 7-11 (9) | 0.90 (0.81) | 1.18 | 0.73 | 1.61 |
| 8 | 9 | 11-15 (13) | 0.69 (0.70) | 0.97 | 0.61 | 1.58 |

Source of the first columns: Skills.txt Charged Bolt row: Param1 3 bolts + Param2 1 more per level, EMin 4, EMax 8, ELev1 1 per level up to level 8 (V, mpq-cat). "per xp-kill" divides cast commands by (xp sum / base xp), because the original's rows hold two merged kills (xp 36 or 54) and 0-xp rows; d2d rows are always exactly one kill (xp 18 or 7, V csv). The per-kill means of the original are therefore slightly low; per xp-kill is the safer base: 1.12 / 0.81 / 0.70.

Analytic expected casts per kill, ceil(HP / (avg damage x bolts hitting)): needs the Fallen's HP, which I could not read (MonLvl.txt not found by mpq-cat, though MonStats gives Fallen Normal MinHP 1, MaxHP 3 as percent columns of MonLvl's life for its level, V; the level and L-HP value were not read). U: a Normal Blood Moor Fallen has roughly 5 to 12 life (memory, not checked). With 3 bolts of average 6 and each bolt hitting a lone target about half the time (the wiggle path, below), one cast kills a Fallen of about 6 life, 1.0 casts: the original's 1.0 to 1.1 commands per kill at CB1 is consistent with that, the d2d's 1.42 casts is a bit higher (not significant, see "residual").

## Ranked causes

1. **Command is not cast (probe artifact), explains CB4 and CB8 completely and about 60 % of CB1.** Mechanism (U, consistent with data): the Fighter's `castGapMs` is 450 ms (internal/env/tuning_combat.go:72), shorter than the 14 tick (560 ms) cast on d2d (docs/d2d-fidelity.md row 7a, V earlier), and d2d lockstep time advances in exact steps, so the next command lands mid-cast and is dropped. On Game.exe the real loop adds latency (screen read, X input), so commands land after the cast and are mostly effective (U: not measured). Check, one line: rerun `d2dcasts` counting casts as mana spent / cost, or with `castGapMs` 650: if commands per kill fall to ~1.4 / 0.7 / 0.6 the gap is closed. Cost 0.3 agent-h (d2d only, hidden, under the lab lock). On the original, count casts from mana (no god) or from the player's cast mode transitions, 0.5 agent-h, needs the live instance (not offline).
2. **Bolt hit rate per cast (engine, residual about 1.42 vs under 1.0 at CB1).** d2d's Charged Bolt flight is the original's wiggle path, checked frame by frame against game.exe (docs/research/re/missiles.md; tests/test_monsters.cpp chargedbolt rows, upstream, S: I read the claims, did not rerun), and the bolt count comes from the skill calc (fight.cpp, `kScatteredBolts` branch, V read). So count and path agree on paper. Check: spawn one Fallen at fixed range, cast CB1 20 times under god, count bolt hits (`hp` drop events) per cast on d2d vs on the original: 1 agent-h, needs the original live; ask the user first.
3. **Monster life or resist difference**: d2d reads life from the same MonStats and MonLvl rows (monsters.hpp `monster_stats`, V read) and Fallen have 0 lightning resist (MonStats, V), so no change from resists. Fidelity doc says monsters are the best covered area (S). Check: `debug`-dump the 20 Fallen lives on d2d (read-only) against the original's memory read of the same area, 0.3 agent-h. Likely not the cause.
4. **Kill attribution**: the original's rows merge 2 to 3 kills and carry 0-xp rows (V csv: 19 of 51 CB1 rows have xp 0, 8 rows have 36 or 54). If a merged row counts one `kill` but its casts cover two monsters the original's per-kill number is biased low by up to 15 percent. Handled in the table (per xp-kill) and it does not move the CB1 gap (1.12 vs 1.42 casts, 2.59 vs 1.12 commands).
5. **Damage or synergy under-applied**: Charged Bolt has no synergy in 1.14d (V, Skills.txt: no calc synergies; Static Field/Lightning do not synergize with it, S from the game's skill descriptions) so there is nothing to under-apply except the per-level step. Not testable from the csv. Low prior.
6. **Hit chance and hit recovery**: lightning spells always hit (a missile that overlaps the unit hits; no attack rating roll, S/U). A hit recovery difference would change how long a Fallen keeps its feet but not casts per kill. Not a cause for casts.
7. **Method biases (ORIGPROBES)**: one 7 minute original run per level (39 to 62 Fallen kills, binomial noise about plus or minus 0.12 on a 0.9 mean), god life and mana on both sides, same Fighter: small next to the 1.8x command-vs-cast factor.

## The bigger point: casts barely drive kills per hour

Seconds per kill row (V csv, 3600 s of d2d virtual time per level assumed from "4 seeds x 15 min", U; original 420 s): CB1 d2d 10.9 s, orig 5.1 s; CB4 11.3 s vs 6.8 s; CB8 10.0 s vs 4.0 s. Total cast commands per second are 0.31 / 0.15 / 0.13 on d2d and 0.38 / 0.16 / 0.20 on the original. A cast is 0.56 s, so casting is about 10 to 17 percent of the time on both sides. d2d is slower per kill row by 1.5 to 2.5x, and that is not from casts: it is walking, target finding, wandering between `Clear` calls (30 s per call, radius 60) and pack density (the d2d spawn table had 26 Fallen, 9 Quill Rats, 19 Zombies in the Blood Moor on seed 1: docs/probes/d2d-spawns.csv, V). So fixing the cast gap would not close the -41 % kills/h and XP/h gap. Compare instead: time between kills split into travel, cast and idle (the Fighter's own log lines) on both backends.

## Recommended next experiment

1. (0.3 agent-h, d2d only, safe) Re-run `cmd/d2dcasts` for CB 1/4/8 with a second counter: casts = mana spent / skill cost, and with `castGapMs` at 450 and 650. Confirms cause 1 and gives casts per kill on d2d directly. Needs the lab lock; hidden d2d only.
2. (0.5 agent-h, offline) Add a per-kill time split (walking, casting, waiting) to both probes from existing Fighter debug logs; this addresses the real gap.
3. (1 agent-h, original game, user decides) a cast counter from player mode on Game.exe and the single-Fallen CB1 hit test of cause 2.

## Upstream PR candidate?

None yet. Dropping a command issued mid-cast is the original's behaviour too, as far as known (U); the difference is the bot's pacing, not an engine bug. If the CB1 residual survives experiment 1, the place to look is the bolt hit test against a unit footprint in upstream `components/game/ai.hpp` `missiles_update` and `components/rules/missiles.hpp` wiggle path (S, names only; the user decides on any PR; nothing posted).

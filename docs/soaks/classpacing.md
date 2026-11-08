# classpacing: measured early-game pace per class (TASK-147)

> status: measured 2026-11-08 (EST), first 90 virtual minutes, lockstep d2d. Tags: V verified live (how stated), S read in code, U unverified. Veteran estimate under comparison: `docs/research/class-early-game.md` section 5 (U).

## Method (V)
Fresh character per class through the normal path (`d2bot play --backend d2d --managed --d2d-class N`, class index per `cmd/d2dsetup/tables.json`), d2d lab binary with lab verbs OFF (no `--lab-verbs`), `--d2d-args "--lockstep"`, `--goals campaign`, seed 7, `nice 19`, one run at a time on the worker host. The leveler table is the class's normal30 table, set with `levelerBuild` in `<data>/config.json` (Paladin `paladin-normal30-v2`, the default). `minute` log lines are one per virtual minute (5 wall minutes give about 90). writes after start: 0 for every run (V: lab verbs off, no `d2cheat` run was started, no memory write tool ran; this is a pace measurement, not a gate run). Characters logged in `docs/probes/test-characters.md`.

Commands (on the worker host, from the synced worktree): `docs/probes/classpacing/run-class.sh TAG CLASS_INDEX 7 RUNFOR [LEVELER_BUILD]`; the table below came from `bar1 0`, `nec1 1`, `ama2 3`, `dru2 5`, `ass2 6`, `pal2 2`, `sor1 4` (`sorceress_normal30`), `sor2 4` (`fallback-lightning-v2`); the 5 minute first runs of ama/dru/ass/pal (tags `*1`) stopped near minute 75 to 89 and gave identical rows up to there, so the same seed is reproducible (V). `docs/probes/classpacing/extract.py` keeps only the minute lines (`minutes/*.jsonl`); `stats.py` prints the table. The full bot logs (1 to 5 MB) stay on the worker host (`~/d2d-lab/classpacing/logs`).

## Pace (V): clvl / XP / kills, deaths 0 for every class at every sample

| Class (table) | v10 | v20 | v30 | v45 | v60 | v90 |
|---|---|---|---|---|---|---|
| Paladin (paladin-normal30-v2) | 3 / 1611 / 70 | 3 / 3309 / 137 | 4 / 5296 / 182 | 5 / 7993 / 255 | 5 / 12149 / 389 | 6 / 20511 / 616 |
| Assassin (assassin_fireblast_normal30) | 2 / 1467 / 63 | 3 / 2451 / 108 | 4 / 4092 / 175 | 4 / 7091 / 270 | 5 / 9202 / 365 | 6 / 15759 / 571 |
| Amazon bow (amazon_bow_normal30) | 2 / 1296 / 58 | 3 / 2370 / 106 | 3 / 3285 / 147 | 4 / 4938 / 224 | 5 / 8330 / 325 | 6 / 14466 / 556 |
| Druid (druid_fire_normal30) | 2 / 1236 / 55 | 3 / 2166 / 93 | 3 / 3264 / 144 | 4 / 4911 / 216 | 4 / 6527 / 275 | 5 / 13977 / 504 |
| Necromancer (necromancer_leveling_normal30) | 2 / 1350 / 59 | 3 / 2403 / 109 | 3 / 3504 / 158 | 4 / 5322 / 238 | 4 / 7267 / 312 | 5 / 8987 / 362 |
| Sorceress (sorceress_normal30) | 2 / 1377 / 63 | 3 / 2562 / 119 | 3 / 3582 / 163 | 4 / 5316 / 239 | 4 / 6518 / 281 | 5 / 9530 / 358 |
| Sorceress (fallback-lightning-v2) | same rows as above to the digit (V: `sor1` and `sor2`) | | | | | |
| Barbarian (barbarian_melee_normal30) | 2 / 1014 / 44 | 3 / 1881 / 83 | 3 / 2460 / 109 | 3 / 3298 / 139 | 3 / 3672 / 156 | 4 / 5543 / 238 |

No class failed to fight: the lowest kill count at minute 10 is 44 (Barbarian). No deaths, no restarts of the character (the "Save and Exit" lines are the bot's own chicken, 4 to 10 per run). Codex/question lines: only the Paladin and Sorceress logs carry `ladder: thought` lines (the unlock ladder; 46 to 69 and 38 to 42) and `question` lines (Paladin 2 to 9, Sorceress 1 to 4); the other tables print none (U whether the ladder is off for them or just silent).

## Against the veteran table (U estimate)
- Estimate: Barbarian, Paladin, Amazon at clvl 5 to 6 after 20 minutes; Assassin/Druid/Necromancer medium; the Sorceress slowest (clvl 4 at about 49, 5 at about 80).
- Measured: nobody has clvl 5 at 20 minutes (all are clvl 3). Clvl 5 comes at 45 (Paladin), 60 (Assassin, Amazon), 90 (Druid, Necromancer, Sorceress); the Barbarian is clvl 4 at 90. The Paladin is the fastest, as predicted (about 2.2 times the Sorceress in XP at 90 minutes, 1.9 times at 60), but the Sorceress is NOT the slowest: the Barbarian is, and the Necromancer and Druid are level with her at 90.
- Sorceress baseline reproduced (V): clvl 4 at minute 45 (the old estimate 49), clvl 5 at 90.
- Why the spread is small and the Barbarian low (S/V from `minutes/*.jsonl`): every class spends most of the first hour in Blood Moor on "Recovery: farming gold" (the Barbarian 68 of 90 minutes in area 2, `bar1`), because gold is below the kit. The pace is therefore set by the recovery loop and the equip failures below, not by the class's damage. The Paladin, Assassin and Amazon pull ahead after they leave Blood Moor for the Den, Cold Plains or Cave (areas 8, 3, 9). The measured numbers are NOT a clean per-class damage ranking and must not be used as the recommended-level table until `recoverypace` and `equipfail` are fixed.

## Minutes by zone (data for the recommended-level table, V, areas by id: 1 Rogue Encampment, 2 Blood Moor, 3 Cold Plains, 8 Den of Evil, 9 Cave level 1, 17 Burial Grounds (S: ids as in the bot logs; names not re-checked live))
Minutes in area at the 90-minute mark are in `minutes/*.jsonl` (field `area`); a class-clean version needs the two fixes first. First measured clvl 5 by area: Paladin Rogue Encampment/Blood Moor (min 45), Assassin Den (60), Amazon Cold Plains (60).

## Proposer class leads (docs only, `internal/decide/proposezone.json` not changed)
Current leads: sorceress +1, paladin 0. Measurement supports Paladin ahead of Sorceress by about one clvl at 60 and 90 minutes, which contradicts a sorceress +1 lead; but the data is one seed per class and confounded by the recovery loop (above), so the file is left as is. Proposal for the owner of the proposer: re-measure after `recoverypace`, then set the leads from the clvl at 60 minutes (Paladin 5, Assassin 5, Amazon 5, Druid/Necromancer/Sorceress 4, Barbarian 3).

## Defects found (cards)
- TASK-182 `equipfail` (high): `leveler: equip retry/failed`, `slot click refused ... stillOnCursor`, `item N did not reach slot torso/rArm` in 7 of 8 runs; the Amazon buys a Short Bow and Arrows, cannot equip, sells the bow and buys it again (ama1/ama2 `chore failed` x15).
- TASK-183 `recoverypace` (medium): Blood Moor recovery farm holds every class (same as follow-up 1 of `docs/palmelee.md`).
- TASK-184 `cpnav` (low): Assassin `leveling area unreachable` Cold Plains (nav would cross an unwanted exit trigger).
Seen, not carded: Might "did not come up" is NOT reproduced (Paladin `combat aura up` 9 times, `tries` 1; V, `pal1`), so the TASK-180 follow-up stays closed for lockstep.

## Not done
The task asks for 2 hours; the brief asked for 90 minutes. Only 90 virtual minutes (clvl by minute 10/20/30/45/60/90) are covered. One seed per class.

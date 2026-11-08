# classpacing2: multi-seed class pace re-measure (panelslive)

> status: measured 2026-11-08 (EST), lockstep d2d, 90 virtual minutes per run (6 wall minutes; a run stops at 55 to 131 virtual minutes depending on host load, so "v90" is minute 90 or the last sample when the run reached 82 or more minutes). Tags: V verified live (how stated), S read in code, U unverified. Compares with `docs/soaks/classpacing.md` (one seed, before the equip fix #427, the recovery preset #429 and the nav fix #432).

## Method (V)
Same recipe as classpacing: fresh character per run through the normal path (`d2bot play --backend d2d --managed --d2d-class N`, lab verbs OFF, `--lockstep`, `--goals campaign`, nice 19, one run at a time on the worker host, load checked before each run; writes after start: 0 for all 24 runs, no `d2cheat` was started). Seeds 7, 11, 13 for the seven classes (tables: paladin-normal30-v2, sorceress_normal30, barbarian_melee_normal30, necromancer_leveling_normal30, amazon_bow_normal30, druid_fire_normal30, assassin_fireblast_normal30), plus one Paladin arm per seed named `palg`. Commands: `labrun` the worker with `docs/probes/panelslive/pace-all.sh "7 11 13"` (calls `run-live.sh TAG CLASS SEED 6m TABLE [old]`); minute lines are in `docs/probes/panelslive/minutes/*.jsonl`; `python3 docs/probes/panelslive/stats2.py docs/probes/panelslive/minutes` prints the tables below. The `play` arm carries the player preset (`playerTuning`, ladder on). The `palg` arm writes the pre-fix values into `tuning.json` (`recoveryEndOnKit` 0, `recoveryStallMin` 20, `recoveryStallGold` 100, `povertySellMin` 0), i.e. the code defaults a gate or soak run keeps; it still runs through `d2bot play`, so it is the gate TUNING, not the gate binary path (U: no ladder-less run was made). Characters logged in `docs/probes/test-characters.md`.

## Result (V): mean (min-max) over seeds 7, 11, 13; clvl then kills
| class | seeds | n | v20 clvl / kills | v45 | v60 | v90 | recovery min | deaths | xp at v90 |
|---|---|---|---|---|---|---|---|---|---|
| ama | 7,11,13 | 95,109,131 | 3.0 (3-3) / 124.7 (121-131) | 3.7 (3-4) / 196.3 (158-224) | 3.7 (3-4) / 245.3 (158-298) | 4.3 (3-5) / 348.0 (158-447) | 15.7 (9-20) | 1 | 8967.0 (3333-12384) |
| ass | 7,11,13 | 98,109,92 | 3.3 (3-4) / 120.7 (102-134) | 4.0 (4-4) / 258.7 (242-283) | 5.0 (5-5) / 361.3 (331-390) | 6.0 (6-6) / 542.3 (529-563) | 11.0 (6-14) | 0 | 17228.3 (15912-19834) |
| bar | 7,11,13 | 97,104,89 | 3.0 (3-3) / 79.7 (77-84) | 3.0 (3-3) / 135.7 (131-144) | 4.0 (4-4) / 179.3 (173-184) | 4.0 (4-4) / 251.7 (239-263) | 22.0 (20-25) | 0 | 5578.7 (5403-5849) |
| dru | 7,11,13 | 93,122,98 | 3.0 (3-3) / 109.3 (101-124) | 4.0 (4-4) / 225.7 (201-250) | 4.0 (4-4) / 277.3 (268-288) | 5.3 (5-6) / 431.3 (368-473) | 11.0 (10-13) | 0 | 13352.7 (11510-15129) |
| nec | 7,11,13 | 83,108,90 | 3.0 (3-3) / 123.3 (118-131) | 4.0 (4-4) / 242.7 (232-248) | 5.0 (5-5) / 332.3 (315-358) | 6.0 (6-6) / 554.0 (529-576) | 17.0 (10-21) | 0 | 17047.3 (15124-19693) |
| pal | 7,11,13 | 87,96,72 | 4.0 (4-4) / 146.3 (127-157) | 4.3 (4-5) / 294.7 (251-349) | 5.0 (5-5) / 389.0 (365-427) | 6.0 (6-6) / 561.0 (548-574) | 10.7 (10-11) | 0 | 18148.5 (15139-21158) |
| palg | 7,11,13 | 55,102,68 | 3.3 (3-4) / 124.7 (91-151) | 4.7 (4-5) / 294.0 (238-342) | 5.0 (5-5) / 366.5 (345-388) | 6.0 (6-6) / 597.0 (597-597) | 28.7 (17-41) | 0 | 18536.0 (18536-18536) |
| sor | 7,11,13 | 102,100,85 | 3.0 (3-3) / 124.7 (123-127) | 4.0 (4-4) / 238.7 (230-249) | 4.0 (4-4) / 300.0 (276-313) | 5.0 (5-5) / 474.7 (425-530) | 14.0 (10-20) | 0 | 13430.7 (12754-14040) |

Per run (clvl/kills/xp at minute 20, 45, 60 and the last sample; minutes in the run; minutes in Recovery):
```
ama 7 3/121/2643 4/224/4967 4/298/6770 5/447/11184 95 18
ama 11 3/122/2829 4/207/4751 4/280/7530 5/439/12384 109 9
ama 13 3/131/2784 3/158/3333 3/158/3333 3/158/3333 131 20
ass 7 3/102/2506 4/251/7160 5/363/10231 6/529/15912 98 13
ass 11 4/134/3944 4/242/6535 5/331/9872 6/535/15939 109 6
ass 13 3/126/2785 4/283/7537 5/390/12245 6/563/19834 92 14
bar 7 3/78/1810 3/132/3042 4/173/3924 4/239/5403 97 25
bar 11 3/77/1758 3/131/3007 4/181/4099 4/263/5849 104 20
bar 13 3/84/1705 3/144/2987 4/184/3983 4/253/5484 89 21
dru 7 3/103/2349 4/201/4650 4/276/7574 5/473/13419 93 10
dru 11 3/124/2691 4/226/5555 4/268/7553 5/368/11510 122 13
dru 13 3/101/2145 4/250/5316 4/288/6339 6/453/15129 98 10
nec 7 3/131/2826 4/248/5563 5/315/8395 6/529/15124 83 10
nec 11 3/118/2694 4/248/6613 5/358/10750 6/557/16325 108 20
nec 13 3/121/2811 4/232/5235 5/324/8641 6/576/19693 90 21
pal 7 4/157/4214 4/284/7246 5/365/9140 6/548/15139 87 11
pal 11 4/127/3822 4/251/7410 5/375/11179 6/574/21158 96 10
pal 13 4/155/4714 5/349/10348 5/427/13738 - 72 11
palg 7 4/151/4444 5/342/11751 - - 55 28
palg 11 3/132/3015 5/302/8405 5/388/11440 6/597/18536 102 41
palg 13 3/91/1963 4/238/6317 5/345/10844 - 68 17
sor 7 3/124/2655 4/249/5424 4/313/7400 5/425/12754 102 12
sor 11 3/123/2799 4/230/5575 4/276/7498 5/469/14040 100 20
sor 13 3/127/2793 4/237/5085 4/311/7169 5/530/13498 85 10
```

Runs shorter than 82 minutes have no v90 (pal s13 72, palg s7 55, palg s13 68); their means use fewer seeds. The "recovery min" column counts minutes labelled "Recovery" within the first 90 (or the whole short run).

## Against classpacing (seed 7 then; V for both sides, one old run per class)
| class | old v90 clvl / kills / xp (seed 7) | new range v90 xp (3 seeds) | verdict |
|---|---|---|---|
| Paladin | 6 / 616 / 20511 | 15139 - 21158 (2 runs with v90) | no clear gain: old inside the range; the gain is early (v20 clvl 4 in all three seeds against 3, kills 146 against 137) |
| Assassin | 6 / 571 / 15759 | 15912 - 19834 | no clear gain (same-seed 15912 vs 15759); seeds 11 and 13 higher but one old seed cannot say |
| Amazon | 6 / 556 / 14466 | 3333 - 12384 | WORSE or broken: seed 13 deadlocked (below); seeds 7 and 11 at clvl 5, 11.2k and 12.4k XP, under the old 14.5k |
| Druid | 5 / 504 / 13977 | 11510 - 15129 | no clear gain (old inside the range) |
| Necromancer | 5 / 362 / 8987 | 15124 - 19693 | CLEAR gain: all three seeds clvl 6, 1.7 to 2.2 times the old XP, same seed 15124 vs 8987 |
| Sorceress | 5 / 358 / 9530 | 12754 - 14040 | CLEAR gain: all three seeds above the old value, +34 to +47 percent; clvl 5 in both |
| Barbarian | 4 / 238 / 5543 | 5403 - 5849 | no gain: identical, still the slowest, still 20 to 25 minutes in Recovery |

Time in Recovery (first 90 minutes): Paladin 10 to 11 (preset), Paladin gate-tuning arm 17 to 41 (mean 28.7), Assassin 6 to 14, Druid 10 to 13, Sorceress 10 to 20, Necromancer 10 to 21, Amazon 9 to 20, Barbarian 20 to 25. The old classpacing note says "most of the first hour" for every class; that is gone for the Paladin, Assassin, Druid and Sorceress, not for the Barbarian.

## Gate-profile confirmation (Paladin)
The `palg` arm (pre-fix tuning values, the gate defaults) spends 17, 41 and 28 minutes in Recovery against 10 to 11 for the preset, and its pace matches the old classpacing Paladin (v45 clvl 4 to 5 and 6.3k to 11.8k XP against 7993; v60 clvl 5, 10.8k to 11.4k against 12149; v90 clvl 6, 18.5k against 20.5k): the code defaults are unchanged and the preset applies only through the ladder (V; also S `TestPlayerTuningPreset`). The preset saved about 18 recovery minutes per run but did not raise the Paladin's XP at v60 or v90 beyond seed noise (palg v90 18536 sits inside the preset's 15139 - 21158): in kills and levels it is equal after 60 minutes.

## Failures and findings
- Amazon seed 13 (V, `ama-s13`): from virtual minute 35 to the end (131 minutes) clvl 3, 158 kills, gold 4, 39 chickens (Save and Exit) in 3 minutes of wall time, 1251 "combat no bound attack skill, swinging the basic attack" and "combat weapon problem, not attacking: thrown weapon stack empty (weapon jav, ammo 0)". The hero has no javelins and no gold for a bow. Card TASK-204 (high). Seeds 7 and 11 also log 500 to 750 "no bound attack skill" lines; they recovered.
- One death: Amazon seed 11 at minute 22 ("Restocking after death"); no other class died in 24 runs.
- Short runs: three runs stopped before minute 82 because the lockstep clock slows when the host is loaded (the user was playing); nothing failed in them.
- No class failed to fight except the Amazon deadlock. The Barbarian is slowest in kills (80 at v20 against 100 to 146 for the others) in all three seeds.

## Is the pace better than classpacing?
Yes for the Necromancer and Sorceress (outside seed noise), early game for the Paladin (clvl 4 at minute 20 on every seed), no clear gain for the Assassin, Druid, Barbarian and Paladin at v90 (ranges overlap), worse for the Amazon. One old seed per class means a "no clear gain" is also "not shown". The proposer class leads (`docs/soaks/classpacing.md`) are unchanged by this data: Paladin ahead of Sorceress by about one clvl at 60 minutes persists (5 against 4 on all seeds), Necromancer now matches the Paladin at 60 (clvl 5).

## Not done
Only 90 virtual minutes; three seeds; the gate arm is the tuning, not a ladder-less binary path; no browser; hardcore and Hell not touched.

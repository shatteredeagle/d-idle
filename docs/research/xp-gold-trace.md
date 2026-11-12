# XPGOLD: the economy of a kill (XP, gold, potions) and why the fresh Paladin was clvl 6 after 60 minutes

> status: research, 2026-11-12, agent decomp14 (queue item 24). Offline: no game, no d2d slot, nothing posted. Builds on three finished traces and adds only what they lack: the per-area table at the bot's real kill rates, the clvl-by-minute prediction, the Experience.txt thresholds, the comparison with the first-hour log. Tags: V read (disassembly), V run (game.exe code under the emulator, compared with an oracle), D (patched 1.14d tables via mpq-cat), S (an earlier trace of ours, named), I inferred, U unverified. Sources: docs/research/xp-award-trace.md (decompxp: the award path), docs/research/monster-stats-trace.md (monsterstats: the real unit populations), docs/research/treasure-trace.md (treasure: classes, gold, potions), docs/probes/hour1-analysis-2026-11-12.md (the hour). Mined helper repos were not used.

## Answer in six lines

1. **The hour is explained by the tables; nothing is wrong with the XP formula.** A Normal Act 1 kill in Blood Moor / Cold Plains / Den pays 22 to 30 XP on average at clvl 1 to 6 (V run, section 3). The observed 17,032 XP over 528 kills is 32 per kill; the model of section 5 on the observed schedule gives 15,880 (clvl 6), 7 % under the observed value.
2. **Even a perfect route does not get close to the SME's 13-15.** Clvl 13 needs 112,725 XP, clvl 15 needs 176,132 (D, Experience.txt). At the best Act 1 field (Cold Plains from minute 15, 14.6 kills per minute) the same model gives 26,000 XP, clvl 7. Clvl 13 would take about 31 kills per minute at 60 XP per kill for the whole hour, or Act 2 and later monsters (section 6). The 13-15 figure is U and not reachable by kills in Act 1 Normal; ask the d2-sme where it comes from (section 9).
3. **Routing loss is real but second order: about 10,000 XP, one level.** The hour analysis found about 25 of 60 minutes lost to bounces, a forced Blood Moor re-clear, a stale quest claim and junk trips. At Cold Plains rates that is worth clvl 7 instead of clvl 6 (section 5). A higher kill rate is worth more than routing: at 25 kills per minute and Stony Field from minute 30 the model gives clvl 9 to 10.
4. **XP per kill is capped by the monster's level, and the cap bites from clvl 7.** Blood Moor (mlvl 1) pays 22.0 up to clvl 6, 17.1 at 7, 13.4 at 8, 4.9 at 10 (V run). The bot's `xpLevelPct` equals game.exe's two tables exactly (B1: no deviation). The bot's area table is right for plain monsters but leaves out champions, uniques and minions, which raise the real per-kill XP by 15 to 50 % in the Den, Cave and Stony Field (B2).
5. **Gold is not the limit and not an early-game lever.** A plain kill drops 0.131 gold piles of `3.5 x mlvl - 0.5` coins: 0.39 gold per kill in Blood Moor, 0.61 Cold Plains, 0.50 Den, 1.4 Stony Field, 2.2 to 2.5 Dark Wood / Black Marsh (V run odds + D). 528 kills are about 300 gold; the hour's gold came from selling items (gems at 250 each). The 34 to 1,007 gold range is what the tables give.
6. **No d2d deviation found; no PR candidate.** The d2d XP path matches game.exe on all 119,790 grid lines at the base that includes upstream 01337ba3; the lab binaries carrying the old formula are a known fork issue (xp-award-trace.md). Gold and drops equal game.exe on 100,000 rolls (treasure-trace.md).

## Questions (Q1..) and where each is answered

| id | question | answer | where | tag |
|---|---|---|---|---|
| Q1 | XP of one kill: formula | stat 13 at spawn = MonLvl XP(level) x MonStats Exp % / 100, then per kill `FUN_0057e480`: level-difference step `FUN_0057e2f0` (tables 0x6e1668 / 0x6e1694, clvl >= 25: exp x clvl / mlvl), ExpRatio `FUN_0057e390` (1024 for levels 1..40, so no change), MaxLvl cap, item_addexperience stat 0x55 | xp-award-trace.md; this doc section 2 | V read + V run |
| Q2 | hero vs monster level gap | 100 % from 5 below to 5 above; 80.9 / 62.1 / 43.0 / 23.8 / 5.1 % for 6 to 10+ levels above the monster; 87.9 / 68.0 / 35.9 / 14.8 / 2.0 % for 6 to 10+ levels below | section 2 | V run (119,790 of 119,790) |
| Q3 | party, pet, merc share | no share in single player; the merc gets its own award from `0057e860`; the hero's award is credited whoever lands the blow (pet, merc), so a merc kill gives the hero full XP (control flow read; constants 0x59 / 0x56 are upstream's) | xp-award-trace.md | V read (not run): treat as I in any claim a PR rests on |
| Q4 | champion / unique / minion multipliers | `leveladd` unique mod: level + 3, exp x 5; champion level - 1 after it, exp 3x (S upstream); in the real populations a champion pays 131 to 288 XP at clvl 1 to 6 (section 3) | monster-stats-trace.md; section 3 | V run (population), S (the multiplier text) |
| Q5 | quest and boss XP | the Den of Evil quest pays a skill point, no XP (S, upstream quests.md); Blood Raven pays her own 181 (mlvl 10) and Andariel 1,282, Duriel 6,007 (D + V run, monster-stats-trace.md section 4); no cleared-area bonus in the award path | xp-award-trace.md | S / V run |
| Q6 | Experience.txt thresholds | XP to reach level n+1 (Paladin, all classes equal): 500, 1,500, 3,750, 7,875, 14,175, 22,680, 32,886, 44,396, 57,715, 72,144, 90,180, 112,725 (level 13), 140,906, 176,132 (level 15) | `internal/telemetry/exp.go` agrees | D |
| Q7 | gold per kill | pile = ilvl + rand(5 ilvl), ilvl = monster level (min 1), mean 3.5 ilvl - 0.5; plain kill 0.131 piles (casters 0.1125), champion 0.6 piles x5, unique and superunique 0 piles; GF scales coins | treasure-trace.md section 4 and 5 | V run + D |
| Q8 | chance and amount by area | table in section 3 (per kill) | section 3 | V run population + V run odds |
| Q9 | potions, scrolls, no-drop in Act 1 Normal | NoDrop 62.5 %, 0.375 drops per kill; potions 0.057 per plain kill (about 1 per 17 kills), 2 per champion, 4 per unique; scrolls 0.015 | treasure-trace.md section 5 | V run |
| Q10 | xp/min and gold/min by area at the bot's rates | section 4 | section 4 | V run + I (the kill-rate input) |
| Q11 | clvl by minute vs 5-6 observed vs 13-15 expected | section 5 and 6 | sections 5, 6 | V run + I |

## 2. The award path in one table (V read, V run)

All addresses are in xp-award-trace.md; the part this task re-ran:

| step | game.exe | tag |
|---|---|---|
| level gap | `FUN_0057e2f0` eax = mlvl, edx = clvl, ecx = exp; `0x6e1668` = 256, 256, 256, 256, 256, 256, 207, 159, 110, 61, 13 (clvl above the monster); `0x6e1694` = 256 x6, 225, 174, 92, 38, 5 (monster above, clvl < 25); clvl >= 25 and mlvl > clvl: `FUN_00483360(exp, clvl, mlvl)` | V run: `tools/emu/xp_award.py grid` (game.exe's own function under the emulator, 119,790 lines) equals `tools/emu/xpg_model.py check` (our port incl. the MulDiv shortcuts) on all 119,790 lines. The first port without the shortcuts of `FUN_00483360` differed on 10,032 lines (exp 0x7fffff only); the shortcut is part of the rule |
| ExpRatio | `FUN_0057e390`, Experience.txt column ExpRatio: 1024 for levels 1..40 (D), shift 10, award unchanged | D + V run (xp-award-trace.md) |
| thresholds | `00611800` / `00611860` level for experience; table in Q6 | D |

## 3. The population and what a kill pays, per area (V run + D)

Source: `tools/emu/ms_run.sh ms_spawn.py 1-12 2,3,4,5,6,8,9,13,17 0` (game.exe's real room population for 12 seeds, Normal; every stat write of `FUN_00573cb0` per unit: level stat 0xc, experience stat 0xd), analysed by `tools/emu/xpg_model.py area|gold` (award() verified in section 2). "Plain" excludes champions, uniques and minions; "all" is the population as spawned. XP per kill is the integer award after the level-gap step; gold and potions per kill use the treasure-trace odds with ilvl = the unit's level and every unit counted as a melee (H2H) class (casters 0.1125 piles instead of 0.131: I, small).

| area (Levels id) | units plain / all | mean level | XP plain at clvl 1..6 | XP all at clvl 1 / 3..6 | at 7 / 8 / 10 (plain) | gold per kill plain / all | potions per kill plain / all |
|---|---|---|---|---|---|---|---|
| Blood Moor (2) | 1163 / 1163 | 1.0 | 22.0 | 22.0 | 17.1 / 13.4 / 4.9 | 0.39 / 0.39 | 0.057 / 0.057 |
| Cold Plains (3) | 2287 / 2377 | 1.5 | 25.6 | 29.9 | 23.4 / 18.6 / 8.8 | 0.61 / 0.64 | 0.057 / 0.098 |
| Stony Field (4) | 1630 / 1830 | 3.1 | 37.3 | 50.9 (clvl 1), 55.8 | 36.8 / 33.6 / 24.1 | 1.36 / 1.51 | 0.057 / 0.130 |
| Dark Wood (5) | 2625 / 2777 | 5.0 | 47.0 | 55.2 (clvl 1), 60.1 | 47.0 / 47.0 / 47.0 | 2.24 / 2.26 | 0.057 / 0.117 |
| Black Marsh (6) | 2259 / 2391 | 5.5 | 52.8 (51.6 at clvl 1) | 57.0 (clvl 1), 65.6 to 66.9 | 52.8 / 52.8 / 52.8 | 2.45 / 2.48 | 0.057 / 0.100 |
| Den of Evil (8) | 849 / 921 | 1.2 | 26.1 | 37.0 | 22.3 / 17.4 / 7.4 | 0.50 / 0.57 | 0.057 / 0.108 |
| Cave 1 (9) | 1352 / 1484 | 2.0 | 35.2 | 51.3 to 52.9 | 34.1 / 28.3 / 15.9 | 0.84 / 0.94 | 0.057 / 0.150 |
| Cave 2 (13) | 430 / 557 | 1.5 | 27.2 | 65.8 to 67.3 | 25.0 / 20.2 / 10.1 | 0.64 / 0.90 | 0.057 / 0.298 |
| Burial Grounds (17) | 447 / 459 | 2.0 | 40.2 | 39.8 to 43.8 | 40.2 / 31.9 / 16.5 | 0.85 / 0.83 | 0.057 / 0.160 |

Reading: (a) a plain kill is worth the same from clvl 1 to 6 in the three starter areas and then falls; the starter areas are exhausted for XP at clvl 7 (Blood Moor 17.1, Den 22.3). (b) The minions and uniques of a pack pay 130 to 310 XP each (Cold Plains minion 131, Stony Field 207, Dark Wood 289, Black Marsh 311) and make "all" 15 to 60 % higher than "plain"; the gap in Cave 2 (27 to 67) is a whole boss pack in a small level. (c) Gold per kill grows with the monster level, 6x from Blood Moor to Black Marsh, but even Black Marsh is 2.5 gold per kill: not a farm (V run + D).

## 4. XP and gold per minute at the bot's real kill rates (V run tables, I rates)

Kill rate input: the bot's own numbers from the hour log (hour1-analysis section 2 and 5): 14.6 kills per minute at Cold Plains, 15 at Blood Moor early, 9.5 Den, 7.4 Blood Moor re-clear, 8.8 over the whole hour. Cells are XP per minute (gold per minute) for plain / all kills at that rate.

| area | rate | clvl 3 | clvl 6 | clvl 8 | clvl 10 |
|---|---|---|---|---|---|
| Blood Moor | 15 | 330 (5.9) | 330 | 201 | 74 |
| Den of Evil | 9.5 | 248 plain, 351 all (4.8) | 248 / 351 | 166 / 275 | 70 / 163 |
| Cold Plains | 14.6 | 374 plain, 437 all (8.9) | 374 / 437 | 272 / 339 | 128 / 196 |
| Stony Field | 14.6 | 545 plain, 815 all (20) | 545 / 815 | 491 / 767 | 352 / 642 |
| Dark Wood | 14.6 | 686 plain, 877 all (33) | 686 / 877 | 686 / 877 | 686 / 877 |
| Black Marsh | 14.6 | 771 plain, 958 all (36) | 771 / 958 | 771 / 958 | 771 / 958 |

Observed on the log (hour1-analysis section 5): Blood Moor fresh 337 per minute (model 330), Cold Plains 558 (model 373 to 436; the gap is I: pack kills or a higher rate in that window), Den 400 first pass (model 248 to 352), Blood Moor re-clear 171 (model 74 to 100 at clvl 5-6: the log has the remaining boss kills; I). The table and the log agree within the composition of what the sweep happened to kill.

## 5. Clvl by minute (model, I for the schedules, V run for the per-kill numbers)

`tools/emu/xpg_model.py sim`: per minute `rate x mean award at the current clvl`; thresholds from Q6.

| schedule | plain kills only | with packs as spawned |
|---|---|---|
| observed-like: Blood Moor 12 min, Den 14, town 6, Blood Moor 13, Den 5, Cold Plains 10 | 13,343 XP, clvl 5 at 60 | 15,880 XP, clvl 6 at 60 (observed: 17,032, clvl 6) |
| ideal route: Blood Moor 5, Den 10, Cold Plains 45 (14.6 kills/min) | 21,971 XP, clvl 6 (clvl 5 at 30, 6 at 40) | 26,020 XP, clvl 7 at 60 |
| 25 kills/min: Blood Moor 4, Den 6, Cold Plains 20, Stony Field 30 | 45,375 XP, clvl 9 | 61,278 XP, clvl 10 |

Clvl at minutes 10 / 20 / 30 / 40 / 50 / 60 for the observed-like schedule (with packs): 3, 4, 4, 5, 5, 6, against the observed log (clvl 2 at 02:20, 3 at 02:23, 4 at 02:36, 5 at 02:45, 6 at 03:11 from a 02:18 start: 3 / 4 / 4-5 / 5 / 5 / 6 at the same minutes). The model reproduces the observed curve.

**Where the gap to 13-15 sits.** Observed 17,032 to ideal-route 26,000: routing, 1.5x. Ideal-route to 112,725 (clvl 13): 4.3x more, of which the kill rate (14.6 now; the model's 25 per minute gives 2.4x XP) and the area (Stony Field and later pay 1.5 to 2.5x per kill, and Dark Wood / Black Marsh are not penalised before clvl 11) are the levers; neither is the formula. The formula only decides the cap (section 3).

## 6. What a human would do to be clvl 13 in an hour (S/I, U)

Reaching clvl 13 needs 112,725 XP. Per-kill XP in Normal: Cave / Crypt / Burial plain 35 to 40, Black Marsh 53, Outer Cloister (Act 1 Monastery, mlvl 8) 76, Catacombs 2 97, Rocky Waste (Act 2) 159, Lost City 219, Sewers 140 (monster-stats-trace.md section 4, V run). No Act 1 area pays more than 76 per plain kill (Outer Cloister); Act 2 areas at 140 to 220 per plain kill need 500 to 800 kills for 112,725. So a clvl 13 hour requires Act 2 (about 700 kills at 160), a kill rate above 12 per minute **and** being in Act 2 by minute 15: not a Normal Act 1 outcome. U: the d2-sme figure (13-15) may count Nightmare, a better build, a party (the XP of one kill is not shared in single player, but a /players 8 game multiplies the monster's experience: 50 % per player from the third, `00573910`, S) or a different time basis. Ask it (section 9).

## 7. Deviations (D1..) d2d vs game.exe

None proven in XP, gold or the Act 1/2 treasure.

| id | game.exe | d2d | tag | differential |
|---|---|---|---|---|
| D1 | `FUN_0057e480` full award path (tables, clvl 25 rule, MulDiv shortcuts) | upstream >= 01337ba3 ports it (400 oracle asserts); lab binaries on e36922ea still carry the old `kill_exp` (37,288 of 119,790 lines differ) | S + V run (xp-award-trace.md); this run re-confirms the game side: our port equals the game on 119,790 of 119,790 | exists: `tools/emu/xp_award.py d2d-old` (old) and upstream's `tools/emu/kill_exp.py` (new); a d2d build >= 01337ba3 should give 0. Fork rebase is the owner's action, not a PR |
| D2 | minion and unique experience: leveladd x 5 on the boss and the minions' own level / exp (the population above shows minion exp 131 to 311) | not read on d2d's side (`boss_stats`) in this task | U | build: d2d's `rules::boss_stats` on the same room population; compare stat 13 per unit with `ms_spawn.py` output; fits monster-stats-trace.md's D-rows (that trace found boss stats read at the boosted level) |
| D3 | merc and pet kills credit the owner (Q3), merc share x 0x56 / 256 | upstream ports `merc_kill_exp` (S) | S, not run | needs a pet / merc killer hook in a `xp_award.py`-style harness (FUN_0057e990 with a stubbed unit) |
| D4 | gold, potions, scrolls: NoDrop 62.5 %, piles `ilvl + rand(5 ilvl)` | equal on 100,000 rolls and 39,975 of 39,980 items | V run (treasure-trace.md) | exists (`tre_diff.py`) |

## 8. Bot findings (B1..), described, NOT changed

| id | where | what | rule | tag |
|---|---|---|---|---|
| B1 | `internal/bot/recoveryxp.go:34` `xpLevelPct` | the level-gap percentages 87.9 / 68.0 / 35.9 / 14.8 / 1.95 above and 80.9 / 62.1 / 43.0 / 23.8 / 5.08 below, clvl >= 25 rule | equals `0x6e1694` / `0x6e1668` / `FUN_0057e2f0`; no deviation (kept as an explicit negative finding) | V run |
| B2 | `internal/bot/recoveryxp.go:44` `areaXPPerKill`, `:66` `areaXPTable` | per-kill XP of an area from the PLAIN-monster mean (Blood Moor 22.0, Cold Plains 25.6, Den 26.1, Stony 37.3, Dark Wood 47.0) and `row.mlvl` rounded; no champions, uniques or minions; the level gap is applied to the mean level, not per unit | the real population is 15 to 60 % richer per kill ("all" columns of section 3, e.g. Stony Field 37.3 plain vs 55.8 all, Cave 2 27 vs 67), and the penalty is applied to the mean, not per unit (a few percent for ranking); the plain-only mean under-ranks pack-rich areas (Stony Field, Cave 2, Cave 1) | V run |
| B3 | `internal/bot/recoveryxp.go:11-19` `recoveryXPTier`, `normalMLvl` | farm tier 2,3,4,5,6,9,13; mlvl 1,2,4,5,6,... | consistent with the table above; the rank by XP would put Stony Field (4) above Cold Plains from clvl 3 (545 vs 374 plain XP per minute at 14.6 kills/min) and Dark Wood above Stony from clvl 6; the hour kept the hero at Blood Moor / Den / Cold Plains (hour1-analysis section 3), which is the planner's "window" data, not this ranking (U which code chose it) | V run (table), U (planner) |
| B4 | `internal/data/supply.go` `GoldPerKill`, `PileAtLeast` | already the treasure-trace numbers (0.39 Blood Moor) | equals the table; the old `GoldPerKill 6` of treasure-trace B1 is fixed here; `internal/resource/sim.go:422` uses it | V run |
| B5 | `docs/probes/hour1-analysis-2026-11-12.md` section 1 | cites the D2 XP table as "500 / 1500 / 3750 / 7500 / 14000" | Experience.txt is 500 / 1,500 / 3,750 / 7,875 / 14,175 (D; `internal/telemetry/exp.go` has the right numbers); the levels in that doc are unchanged | D |
| B6 | the hour's SME comparison "clvl 13 to 15" (brief, not in a doc) | assumed reachable from Act 1 Normal kills | needs 112,725 to 176,132 XP; the best Act 1 plain area pays 53 XP per kill; see sections 5 and 6 | V run, U for the source of the number |
| B7 | `docs/probes/hour1-analysis-2026-11-12.md` section 1 | "634 combat kill lines" vs 528 `minute` kills | not explained there; xp-award-trace.md already notes credited 0-XP kills (about one row in six on the live original). Not resolved here | U |

## 9. PR candidates, drafts

**No PR candidate.** The only XP defect found in d2d (old `kill_exp`) is already fixed upstream (01337ba3); gold and drops equal game.exe on 100,000 rolls. Nothing was built in the fork checkout; nothing was added to docs/upstream/pr-queue.md.

Questions for the d2-sme (the answer decides whether anything is wrong with the expectation): (1) which areas, clvl at which minute, and how many kills did the 13-15 figure assume? (2) Is it Normal single player? (3) Does it count Nightmare, Act 2 (Radament, Sewers) or a /players 8 game? (4) What kill rate per minute does a human Paladin reach in Cold Plains?

## 10. Drill recipe for the fork verbs (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md; run later, not here)

- **X1 per-kill XP.** Spawn `fallen1` and `zombie1` at clvl 3 and 8 on a throwaway character (`debug spawn`, kill, read the stat 13 delta). Expect 18 / 33 at clvl 3 and, at clvl 8 (mlvl 1, difference 7, table entry 159 of 256), 18 x 159 / 256 = 11 and 33 x 159 / 256 = 20. A d2d on the old `kill_exp` differs from clvl 7. U until run.
- **X2 pack kills.** Kill a Cold Plains champion and a unique's minions at clvl 3: expect 131 to 160 per minion / unique (section 3). Settles D2.
- **X3 gold by area.** Kill 2,000 Blood Moor and 2,000 Dark Wood plain monsters: gold per kill 0.39 vs 2.2 (treasure-trace T4).
- **X4 merc credit.** Hire a merc, let it land the killing blow on a Fallen: the hero's XP rises by the full 18 (Q3), the merc by its own award.

## 11. Matrix cell updates (proposed text)

- docs/classes/ranking.md and the paladin plan docs: "an hour of Normal Act 1 is clvl 6 (17,000 XP) at 8.8 kills per minute and clvl 7 at best (26,000 XP); clvl 9 to 10 needs about 25 kills per minute and Stony Field from minute 30. Clvl 13 is Act 2 territory."
- docs/research/monster-stats-trace.md B1: add that the plain-only mean under-counts champions, uniques and minions by 15 to 60 % (this doc, section 3).
- hour1-analysis section 1: correct the thresholds (B5).

## 12. Dead ends and limits

- The number of kills the real bot makes of each type is not logged (hour1-analysis, xp-award-trace.md): the "plain" and "all" columns bracket it; the true mix lies between.
- The caster gold and potion odds (Cast / Miss classes) were not split per unit; every unit counted as H2H (I, error under 15 % of the gold, which is small).
- Superunique kills (Bishibosh, Corpsefire, Blood Raven) are not in the population rows (flag 2 does not occur in the room populations; they are placed by their own spawn path). Corpsefire (mlvl 4 + leveladd, exp x 5) and Blood Raven (181) matter for the first levels but are one kill each.
- Not run: any live game, any d2d binary. The D2 deviation (boss exp in d2d) needs the build.
- Not checked: the hour's 15 of 60 minutes of town and routing losses, owned by the hour1-analysis cards.

## Tools

- `tools/emu/xpg_model.py` (`check`, `area`, `gold`, `sim`): the port of the award function (verified on the game grid), the population analysis, the gold / potion per kill and the clvl-by-minute model. Inputs from the worker host: `bash tools/emu/ms_run.sh xp_award.py grid` (119,790 lines, 40 s) and `bash tools/emu/ms_run.sh ms_spawn.py 1-12 2,3,4,5,6,8,9,13,17 0` (25 s for four seeds, 75 s for twelve; Normal; the lines starting with `U`).

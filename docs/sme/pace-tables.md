# Pace tables: human and speedrun reference for soak milestones (classic LoD 1.14d, offline single player)

Author: SMEKB-E, 2026-10-29. Tags: **V** fetched primary or reliable source (URL given), **S** snippet or secondary, **U** unverified or from memory (judgment),
**C** computed by us from V inputs, **B** our own bot data (repo doc). `[D2R]` marks D2R-only facts; 1.14d is the target.

Honest summary first: **no source I could fetch gives measured XP/hour, gold/hour or "clvl after N hours" for 1.14d.** Everything in the pace columns is U (veteran
judgment) except the XP spans (C from V table), the best-area-per-level chart (V, v1.10) and our own bot telemetry (B). Use the tables as order-of-magnitude
yardsticks, then replace them with measured values from the bot's calibration logs. Do not treat any U cell as a pass/fail threshold on its own.

## Sources
- XPT = XP table, monster level and area level data: `docs/sme/mechanics-progression.md` (V there: Amazon Basin Experience page, diablowiki area-level table via Wayback).
- LUG = PureDiablo "Level Up Guide v1.10", best area per clvl: https://www.purediablo.com/strategy/diablo-2-guide-level-up-guide-v1-10/ (V, fetched; 1.10 data, XP formulas for the listed levels are the same family as 1.14d but values "differ greatly in earlier versions").
- ALM = Almar's guide, route by level: https://almarsguides.com/Computer/Games/Diablo2/Leveling/ (V, old 1.10-era, no hours stated).
- DBS = DiabloBytes speedrun walkthrough, level checkpoints: https://www.diablobytes.com/diablo-ii/guides/speedrun/walkthrough/ (V text, **[D2R] speedrun community**).
- SRC = speedrun.com Diablo II (Resurrected) Normal Any% Baal: Sorceress P8 softcore 53m36s, P1 hardcore 52m41s, patch 2.4+ (S, search snippets of user pages e.g. https://www.speedrun.com/users/NoviceCho; **[D2R] only**, no 1.14d category seen).
- OUR = our bot data: `docs/soak-analysis.md` (Soaky run, B), `docs/g1-log.md` (B), `docs/sme/economy-data.md` (gold, mostly U), `docs/soak24.md` (current XP floor 1000 XP/h).
- Searches for forum "clvl at Andariel/Duriel/Mephisto" and "hours to beat Normal" returned only D3/D4/D2R pages (S, unusable); HowLongToBeat quotes ~14 h rushed / ~21 h main for D2R (S, [D2R], whole game incl. Hell excluded). Not used for numbers.

## 1. XP spans and kills needed (C, from XPT table)

Total XP to clvl 5/10/15/20/25/30/35/40 = 7,875 / 57,715 / 176,132 / 537,513 / 1,640,359 / 4,663,553 / 9,629,723 / 17,270,791.

| clvl band | XP span (C) | kills needed at 20 XP/kill | at 100 | at 500 | at 2,000 |
|---|---|---|---|---|---|
| 1-10 | 57,715 | 2,886 | 577 | 115 | 29 |
| 10-15 | 118,417 | 5,921 | 1,184 | 237 | 59 |
| 15-20 | 361,381 | 18,069 | 3,614 | 723 | 181 |
| 20-25 | 1,102,846 | 55,142 | 11,028 | 2,206 | 551 |
| 25-30 | 3,023,194 | 151,160 | 30,232 | 6,046 | 1,512 |
| 30-35 | 4,966,170 | 248,309 | 49,662 | 9,932 | 2,483 |
| 35-40 | 7,641,068 | 382,053 | 76,411 | 15,282 | 3,821 |

Kills per hour to finish a band in N hours = span / (N * XP per kill). Two anchors for XP per kill:
- B: Soaky in the Den of Evil at clvl 4-6 averaged **18 XP/kill, about 1,000 kills/h**, 13-18k XP/h (578 kills, 10,435 XP in 2,106 s, `docs/soak-analysis.md` section 2). A human with a decent skill clears about the same kill rate there; the gap is time lost elsewhere (potion runs, pauses, re-entering fresh games), see section 5.
- C/V: the per-kill XP formula is `mlvl * MonStats.exp / 100` times the difficulty XP multipliers, then the level-difference penalty (XPT section 1-2). The PureDiablo chart lists average base XP per kill from 78 (Normal Act 1) to 56,680 (Hell Act 5), a ratio of about 730 (V LUG, 1.10; absolute values are of the average monster in the chart's own scale, treat the ratio as the useful number).
- U: so XP/kill at a sensible "right area for your level" runs roughly 20-60 (Normal Act 1, clvl 5-10), 100-300 (Act 2-3 Normal, clvl 15-24), 400-1,500 (Act 4/5 Normal clvl 25-35), 1,500-4,000 (Nightmare, clvl 36-45). These are consistent with a human needing "600-1,200 kills/h" and landing at the XP/h in section 2. No source fetched to confirm; recompute from `monstats.txt` if the bot has the data (docs/data.md) before relying on them.

## 2. Best area per band and XP/hour (V for areas, U for rates)

### 2.1 Where humans level (V LUG, v1.10, "area/difficulty with maximum experience per kill, assuming one monster of each type")

| clvl | best area and difficulty |
|---|---|
| 1-11 | Normal Act 1 |
| 12-18 | Normal Act 2 |
| 19-23 | Normal Act 3 |
| 24-31 | Normal Act 4 |
| 32-36 | Normal Act 5 |
| 37-43 | Nightmare Act 1 |
| 44-48 | Nightmare Act 2 |
| 49-52 | Nightmare Act 3 |
| 53-62 | Nightmare Act 4 |
| 63-73 | Hell Act 1 |
| 74-80 | Hell Act 2 |

The page warns it is "not iron-clad": a faster-killing area with a lower XP percentage can beat it, and item find matters. Almar (V ALM): Normal 1-15 Tristram runs, 15-24 Tomb of Tal Rasha (Act 2) runs, 25-40 Chaos Sanctuary/Baal runs; Nightmare 41-63 Chaos/Baal; Hell 64-90 Chaos/Baal. This is a fast-route twinked-friendly guide; a beginner caster should follow LUG instead.

### 2.2 Concrete areas, with monster levels from XPT section 5.1 (Normal/Nightmare)

| clvl | Normal areas (mlvl) | notes for a caster / melee |
|---|---|---|
| 1-5 | Blood Moor 1, Cold Plains 2, Den of Evil 1 (quest) | Den: our bot's best at 13-18k XP/h (B). Cold Plains killed Soaky's low-level sorc via chicken loops (B) |
| 5-10 | Den, Cave 1/2 (2), Stony Field 4, Underground Passage 1 (4), Dark Wood 5, Burial Grounds 3 (Blood Raven mlvl 10 boss), Tristram 6, Black Marsh 6 | Humans: quests and Tristram (U, ALM) |
| 10-15 | Tower Cellar 1-5 (7) + Countess, Outer Cloister/Barracks 9, Jail 10, Cathedral/Catacombs 1-2 (11), Andariel's Catacombs 4 (12), Andariel 12 | Catacombs/Cathedral and Andariel: caster Cold/Fire. Andariel has poison and is Normal mlvl 12 (XPT) |
| 15-20 | Act 2: Rocky Waste 14, Dry Hills 15, Far Oasis 16, Lost City 17, Valley of Snakes 18, Maggot Lair 17, Tal Rasha's Tomb 17 | Tomb runs (ALM); Duriel mlvl 22 |
| 20-25 | Act 3: Spider Forest 21, Flayer Jungle 22, Kurast Bazaar 22, Travincal 24, Durance 25 | Mephisto 24ish; fast-moving packs, Flayer Jungle for XP (ALM) |
| 25-30 | Act 4: Outer Steppes 26, Plains of Despair 26, City of the Damned 27, River of Flame 27, Chaos Sanctuary 28 | Chaos Sanctuary is the standard (ALM, LUG) but needs a strong kill speed and resists; Izual/Hellforge quests |
| 30-35 | Normal Act 5: Bloody Foothills 24, Frigid Highlands 25, Arreat Plateau 26, Crystalline Passage 29, Glacial Trail 29, Ancients' Way 29, Nihlathak's Halls 32-36, WSK 39-43, Throne 43 | Normal Act 5 "is good XP until clvl ~40" (V Arreat via XPT section 6); Baal ~mlvl 60 in Normal (XPT 5, U which table) |
| 35-40 | Same Act 5 Normal, or start Nightmare Act 1 (mlvl 36-43) | Nightmare Act 1 is 36-43: level-difference penalty is mild above 25 (clvl/mlvl, XPT 2.2) |

Nightmare after Baal: LUG says Nightmare Act 1 from 37, Act 2 from 44, Act 3 from 49, Act 4 from 53 (V). Nightmare also needs a resist plan (-40, XPT 6).

### 2.3 Rates (U, human veteran, 1.14d, an average of caster and melee with decent gear)

| clvl band | difficulty and area | kills/h (U) | XP/h (U) | gold/h from piles (U) | gold/h from selling (U) |
|---|---|---|---|---|---|
| 1-10 | Normal Act 1 | 700-1,200 | 15-30k (our bot: 6.6k wall, 13-18k/h in the Den, B) | 300-1,000 | 1-3k |
| 10-15 | Normal Act 1-2 | 700-1,000 | 40-100k | 1-2k | 3-6k |
| 15-20 | Normal Act 2 | 600-900 | 120-250k | 2-4k | 5-10k |
| 20-25 | Normal Act 2-3 | 500-800 | 300-600k | 3-6k | 8-15k |
| 25-30 | Normal Act 3-4 | 400-700 | 700k-1.5M | 4-8k | 10-20k |
| 30-35 | Normal Act 4-5 | 400-700 | 1.2-2.5M | 5-10k | 15-30k |
| 35-40 | Normal Act 5 / Nightmare Act 1 | 400-700 | 2-4M | 8-20k | 20-40k |

How these were sanity-checked (C): a band from 20 to 25 needs 1.10M XP (section 1), which at 450k XP/h is about 2.5 h; 25-30 needs 3.0M, at 1M XP/h about 3 h. Those match the
veteran sense that a casual first Normal run takes about 8-10 h to reach Baal at clvl ~30. The gold columns are weaker: economy-data.md could not verify gold per hour at any level
(`docs/sme/economy-data.md` section 5) and all gold values are 10x-uncertain. Gold piles scale with mlvl and Gold Find (V diablowiki Gold Find, in economy-data.md); selling magic/rare items beats piles 3-10x
at low levels (S/U there). D2R-only: Terror Zones, shared stash and Gold-find changes do not exist in 1.14d.

Nightmare and Hell XP/h are not tabulated: in NM the level-difference penalty is the same (clvl/mlvl above 25), monsters of mlvl 36-65 give about 2-10x the Normal XP per kill at the same
kills/h, and LUG shows the best XP/kill in NM Act 4 by clvl 53-62. A bot should reach NM only after Normal Act 5 is clean (see section 5 gates); exact NM XP/h is U.

## 3. Human-pace timeline (U unless noted)

Offline single player with "/players 1" style play. Hours are play hours with town trips. "Human" means a competent but not optimised player on a first-pass route (quests that pay, Tristram/Countess/Tomb runs for XP).

| playtime | Sorceress / Necromancer / Druid (caster) | Barbarian / Paladin / Amazon / Assassin (melee/hybrid) |
|---|---|---|
| 2 h | clvl 12-16. Act 1 complete: Den, Cain skipped or done, Andariel dead or being farmed (Andariel mlvl 12). Act 2 just opened. Gold 2-6k net | clvl 11-14. Den, Blood Raven (merc), Countess maybe, Andariel dead or the next target. Act 2 entry |
| 8 h | clvl 26-30. Act 4 Mephisto/Diablo done, Act 5 started (Ancients before Baal at clvl 20+ gated, V S in XPT 6). Normal almost complete. Gold 30-80k incl. selling | clvl 24-28, same acts, a little slower in Act 3 |
| 24 h | clvl 40-48 in Nightmare Act 2-3 (Normal done around hour 8-10, Nightmare Acts 1-2 by hour 20). Gear: 20-40 resist, a runeword in progress | clvl 38-45, Nightmare Act 1-3 |

Speedrun-pace (not a human play pace; routes skip most quests and level only by boss kills):
- [D2R] Normal Any% Baal world-class times are about 53 minutes on a Sorceress (S SRC). The bot cannot match this: it needs perfect teleport routes and rushes.
- [D2R] The DiabloBytes speedrun guide's recommended checkpoints for the Normal sorc run: Forgotten Tower clvl 5, Andariel fight clvl 12, Arcane Sanctuary 16, Duriel 17, Travincal 19, Mephisto "almost 20" (V DBS). So Andariel at clvl 12 and Mephisto at clvl 20 define a **speedrun floor**: a character below these levels at those bosses is either under-leveled or the run is using skips.
- Speedrun at 2 h: past Act 5 Baal (1 h) and into repeated Baal/Hell runs, clvl 35-45 (U, because a Baal run in 1.14d Normal gives about 30-35 on the run alone; fits LUG's Chaos/Baal rows). At 8 h and 24 h a speed runner is in Hell, clvl 70-85 (U). Not a useful bot target.

## 4. Quest and act timestamps, human vs half-human (U)

| step | human (caster) | half human pace | notes |
|---|---|---|---|
| Den of Evil | 0:20-0:45 | 0:40-1:30 | Skill point plus respec; killing the Den gives about 1000 kills/h XP at clvl 1-6 |
| Blood Raven | 0:45-1:30, merc | skip OK | mlvl 10 boss, fire arrows; Normal; **not on the critical path** (U, play-act-route.md); our bot lost 189 attempts underleveled (B R1). Skip or kill only at clvl >= 8-10 with potions |
| Cain | 0:45-1:45 or skip | skip | free identify; not required |
| Andariel | 1:30-2:30 (clvl 14-17) | 3:00-5:00 (clvl 12-16) | gate for Act 2. Poison: carry antidotes. Our bot held at clvl 22 for Andariel in the audit (backlog), a too-cautious threshold |
| Act 2 entry | 1:45-2:45 | 3:30-5:30 | |
| Radament | 2:30-3:30 | optional | +1 skill point; sewers |
| Staff and Summoner | 3:30-5:00 | 8:00-10:00 | Cube, Shaft, Amulet |
| Duriel | 4:00-5:30 (clvl 21-24) | 9:00-12:00 (clvl 18-22) | Act 3 gate; mlvl 22, Cold resist 50 (play-act-route); speedrun checkpoint clvl 17 (V DBS D2R) |
| Act 3 Travincal / Mephisto | 5:30-7:00 | 12:00-16:00 | Khalim's Will chain; Mephisto mlvl 24ish |
| Diablo (Act 4) | 7:00-8:30 | 15:00-20:00 | clvl 26-30 |
| Baal (Normal done) | 8:00-10:00 | 20:00-26:00 | Nightmare unlocks |

## 5. Proposed bot-soak milestones

Principles: (1) milestones use a quest-progress curve against wall-clock hours of the soak, with two bars, "floor" (fail below) and "good" (human at half pace or better). (2) Gold is net: measured as gold on hand plus the sell value of stash, minus spending, and
counts only if the bot has potions and a TP stack (no hoarding while broke, see our GOLDBURN issue in g1-log). (3) No-loop rule: no area, boss or quest step may be repeated more than N times without clvl or an item gain: 3 times for Blood Raven and 5 for Andariel without a clvl change, and no "same area, same fresh game" loop longer than 30 minutes with XP/h below the floor (our bot had 189 Blood Raven aborts in one run, B). (4) A level hold (waiting to level) is allowed only if XP/h stays above the soak floor of 1,000 XP/h (docs/soak24.md) and the area is changed when XP/h falls below it.

### 5.1 What a bot at half human pace reaches

Human timeline from section 3 stretched by 2 (U): 2 h wall = human 1 h (clvl 8-10, Den done, Blood Raven pending, Cold Plains/Stony Field). 8 h wall = human 4 h (clvl 18-22, Andariel dead, in Act 2). 24 h wall = human 12 h (clvl 30-34, Normal done and Nightmare Act 1-2 begun). 72 h wall = human 36 h (clvl 50ish, Nightmare Act 3-4). The bot's own record: clvl 7 after 3.5 h (B), about a quarter of human pace, with 189 wasted Blood Raven attempts and a 25-minute pause flap fixed since then (see BACKLOG). Half human pace is a stretch goal for the 2 h and 8 h gates but a fair expectation for 24 h once the known loops are fixed.

### 5.2 Table (floor = fail below; good = half-human)

| soak time | clvl floor | clvl good | quests done by then (good) | gold net (good, U) | XP/h floor (U) |
|---|---|---|---|---|---|
| 2 h | 7 | 9-10 | Den done; Blood Raven skipped or killed at clvl >= 8; in Burial Grounds/Cave/Tristram | >= 500 and a TP + 4 potions | 8k |
| 4 h | 11 | 12-14 | + Tower Cellar or Countess if clvl >= 12; Andariel attempted at clvl >= 12 with antidotes | >= 1k | 15k |
| 8 h | 16 | 18-22 | Andariel dead, Act 2 reached, Radament optional | >= 3k | 40k |
| 12 h | 20 | 24-26 | Staff parts in progress, Summoner killed | >= 5k | 80k |
| 24 h | 26 | 30-34 | Duriel dead (clvl >= 20), Act 3 reached, Travincal, Mephisto dead if time | >= 10k | 150k |
| 48 h | 34 | 40-44 | Normal complete or Diablo dead; at least one build milestone (resist 50+ for NM) | >= 20k | 400k |
| 72 h | 40 | 46-52 | Nightmare entered with resists; Nightmare Act 1-3 | >= 40k | 800k |

Rules attached:
- Gate to Act 2 (Andariel): clvl >= 12 minimum (speedrun floor, V DBS [D2R]) and >= 16 preferred (human). Our current audit finding of "hold at clvl 22 for Andariel" is far above the human norm of 14-17 and explains stalled soaks; recommend 14 with resists, antidotes and 4+ potions (U).
- Gate to Duriel: clvl >= 18 minimum (speedrun checkpoint 17, V DBS D2R), 20-24 preferred (U), Cold resist or potions (Duriel Cold 50 in Normal, play-act-route).
- Gate to Mephisto: clvl >= 20 minimum (speedrun checkpoint "almost 20", V DBS D2R), 24+ preferred; Diablo clvl >= 24; Baal clvl 26-30 (Ancients quest gate clvl 20 Normal, S in XPT 6).
- Nightmare: clvl >= 30 preferred (LUG recommends Normal Act 5 until clvl 36 for XP, V; our own bot gate in `docs/nightmare-campaign.md`); the human guides say Normal to 40-45 before Hell prep, not before NM (XPT 7 S).
- Soak failure codes to add to d2soak: `below_floor_clvl` (clvl under the floor at a checkpoint), `loop_no_gain` (same area or boss attempted > N with no XP or item gain), `gold_hoard_broke` (net gold above 2k but potions 0 in town for 2 visits).

## 6. Open questions, to measure instead of guess
1. Actual XP/kill by area and difficulty from the bot's own telemetry: log (area, mlvl, kills, XP delta) in the minute summary (soak-analysis WP8) and replace section 2.3.
2. Gold per hour: log pile size by area and sell value per item (economy-data.md calibration logger); until then gold targets are U.
3. A 1.14d human timing source (a video, a ladder-era forum thread, a Wayback copy of a Lurker Lounge or Diabloii.net "how long did your first Normal take" thread). I found none by search; a Wayback sweep of diabloii.net would be the next step.
4. Whether Normal monster levels in XPT (table is area level, Normal monsters have their own type level near it) change the Andariel/Duriel entry numbers by 1-2.
5. Our own `milestones.json` (docs/milestones) has gates by feature, not by clvl/time; the table in 5.2 could be added there as soak criteria once the main agent agrees.

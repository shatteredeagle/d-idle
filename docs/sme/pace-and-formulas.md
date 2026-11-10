# D2 LoD 1.14d pace and formula lookup (Normal Act 1 to Act 2 gate; Paladin and Sorceress)

Backlog TASK-252 (card smepace), 2026-11-10. Source of the ask: `docs/research/project-review-2026-11-10.md` sections 4.4 and 6 (the last d2-sme consult cost about 344k tokens for 450 words, mostly wiki fetching).

## How agents use this page
1. Read this page BEFORE consulting d2-sme. Consult only for a question this page does not answer, and quote the section you already read in the brief.
2. Facts are tagged **SRC** (a primary or reliable page/table, URL or repo table named) or **EXP** (veteran judgment, the d2-sme agent's view; not a fact, never a pass/fail threshold on its own). `[D2R]` = D2R-only, not for 1.14d. Verified-now fetches of 2026-11-10 are marked `(fetched)`; the rest are carried from the repo's SME files (named in each row) which carry their own V/S/U tags.
3. Do not copy numbers out of here into code or contracts. Numbers with an owner stay with the owner: XP table and penalties `docs/sme/mechanics-progression.md`, knobs `docs/knobs.md`, gates in the specs. This page is a lookup, not a decision (decisions: `docs/decisions.md`).
4. A cell tagged EXP that matters becomes a measured number only through our own telemetry (`docs/sme/pace-tables.md` section 6). Until then say EXP in the report.

Sources used (all SRC unless stated):
- AB = Amazon Basin wiki, https://www.theamazonbasin.com/wiki/index.php?title=Prayer / `Sacrifice` / `Smite` (fetched 2026-11-10 with a browser User-Agent).
- XPT = Amazon Basin Experience page and Arreat Summit experience page, as held in `docs/sme/mechanics-progression.md` sections 1 to 4 (SRC there).
- LUG = PureDiablo Level Up Guide v1.10, https://www.purediablo.com/strategy/diablo-2-guide-level-up-guide-v1-10/ (via `docs/sme/pace-tables.md`; 1.10 data).
- ARR = Arreat Summit mirror (Wayback of classic.battle.net/diablo2exp), via the repo SME files.
- TAB = our own install tables (`skills.txt`, charstats) as read in `docs/sme/class-paladin.md` and `class-sorceress.md` (tag T there).
- OUR = bot data (`docs/soak-analysis.md`, `docs/g1-log.md`), tagged B in the SME files.

## 1. Expected clvl by beat, Normal Act 1 (human; the table the review asked for)

The review 4.4 item 4 (d2-sme consult 2026-11-10). All EXP: no fetched source gives clvl-by-hour for 1.14d (`pace-tables.md` honest summary). Two columns of different meaning; the tables in `pace-tables.md` section 3 use play-hours.

| beat (in order) | clvl, human Paladin (with Rogue merc) | clvl, human Sorceress | note |
|---|---|---|---|
| Cold Plains waypoint | 3 to 5 | 3 to 5 | first waypoint; Blood Moor first |
| Den of Evil cleared, Corpsefire (skill point) | 4 to 6 | 4 to 6 | Den mlvl 1 area, 13 to 18k XP/h for our bot at clvl 4 to 6 (OUR) |
| Blood Raven and the Rogue camp | 8 to 10 | 8 to 10 | Blood Raven is mlvl 10 boss; casters need range, see `blood-raven-caster.md` |
| first unique or set drop | 8 to 15 | 8 to 15 | random; a Codex or return-screen headline, not a clvl gate |
| Cain rescued (Dark Wood, Stone of Jordan) | 9 to 13 | 9 to 13 | free identify; not required for progress |
| Countess (Tower Cellar 5) and first runes | 11 to 14 | 11 to 14 | needs Tower Cellar levels 1 to 5; mlvl 7 area, Countess a super unique |
| Andariel dead, Act 2 open | 14 to 18 (hour 2.5 to 4.5) | 12 to 16 | fast routes 13 to 17 |

Floors a veteran does not call overcautious (EXP, review 4.4 item 1):

| step | with the merc | without the merc | kit |
|---|---|---|---|
| Catacombs (Cathedral to Catacombs 1 to 4, mlvl 11 to 12) | clvl 13 | clvl 14 | 4 to 6 healing, 1 to 2 TP scrolls |
| Andariel (mlvl 12 in Normal, poison) | clvl 14 | clvl 16 | 8 healing, 2 antidote potions |

ADR 0011's 15/17 is within one level of that. The clvl-17 floor is fair only as the no-merc Andariel floor; as a hold before the Burial Grounds it is a wall (review 4.4 item 1; the fix is the `actplan` card, not this page).

Speedrun checkpoints for contrast (V `pace-tables.md`, [D2R] speedrun community): Forgotten Tower clvl 5, Andariel clvl 12, Arcane Sanctuary 16, Duriel 17, Travincal 19, Mephisto about 20. A speedrun floor, not a human pace.

Bot comparison (OUR): clvl 7 after 3.5 h in one run (about a quarter of the human pace); 2.4k XP/h on the sandbox in the review's measure (review 4.4, EXP the cell). About 185k XP to clvl 17 (section 2) is roughly 77 h at 2.4k XP/h against 2 to 5 h at a human 40 to 100k XP/h (computed from SRC table, rates EXP).

## 2. XP table and XP to next level (SRC: XPT, computed spans)

Total XP to reach level L (SRC, Amazon Basin Experience): 2: 500, 3: 1,500, 4: 3,750, 5: 7,875, 6: 14,175, 7: 22,680, 8: 32,886, 9: 44,396, 10: 57,715, 11: 72,144, 12: 90,180, 13: 112,725, 14: 140,906, 15: 176,132, 16: 220,165, 17: 275,207, 18: 344,008, 19: 430,010, 20: 537,513, 25: 1,640,359, 30: 4,663,553, 35: 9,629,723, 40: 17,270,791.

XP needed from level L to L+1 (computed from the above): 1: 500; 2: 1,000; 3: 2,250; 4: 4,125; 5: 6,300; 6: 8,505; 7: 10,206; 8: 11,510; 9: 13,319; 10: 14,429; 11: 18,036; 12: 22,545; 13: 28,181; 14: 35,226; 15: 44,033; 16: 55,042; 17: 68,801; 18: 86,002; 19: 107,503.

Kill counts: XP per kill at a sensible area is about 20 to 60 in Normal Act 1 clvl 5 to 10 (EXP; Den measured 18 XP per kill, 1,000 kills/h, B in `pace-tables.md`). To clvl 10 needs 57,715 XP, about 2,900 kills at 20 XP.

XP per kill (SRC `mechanics-progression.md` section 1): `monster level * MonStats.exp / 100`, times the player-count factor, then the level-difference penalty, then % Experience bonuses. Champion/Ghostly/Possessed x3, unique/minion x5 (their mlvl +2 / +3).

## 3. XP penalties (SRC: XPT)

### Level difference, clvl below 25 (`mlvl - clvl`), percent of base XP
| +10 or more | +9 | +8 | +7 | +6 | -5 to +5 | -6 | -7 | -8 | -9 | -10 or lower |
|---|---|---|---|---|---|---|---|---|---|---|
| 1.95 | 14.8 | 35.9 | 68.0 | 87.9 | 100 | 80.9 | 62.1 | 43.0 | 23.8 | 5.08 |

Above clvl 25: monsters higher than you give XP times clvl/mlvl; lower side uses the same table as above (100 percent down to -5, then 80.9, 62.1, 43.0, 23.8, 5.08). Level 70+ takes an extra flat penalty (95.31 percent at 70, falling; not relevant for the early game). Normal difficulty death costs no XP (0 / 5 / 10 percent of the current level span by difficulty, SRC).

### Party and player count (SRC, via `mechanics-progression.md` section 3)
- Players setting 1 to 8 multiplies XP by x1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5 (`(n+1)/2`), and monster HP scales with it, so XP per HP is about constant.
- Party bonus since 1.11: `(members-1) * xp * 87/256`: x1.34 for 2, x1.68 for 3, x2.02 for 4. Members must share the area and be within about 53 yards. Then the pool is split by level (your share = total * yourLevel / sum of levels). Each member's own level-difference penalty applies, so low levels cannot leech high-level kills.
- No XP for mercs and pets themselves; their kills credit you. Killing blow takes the XP if neutral players hit the same monster.
- Practical for us: solo, offline. The party rules do not apply to the single-player bot.

## 4. Monster level vs XP gain; where to farm by clvl

- The one-line rule (EXP, confirmed by the table above): farm where the monster level is at least clvl minus 5. Inside -5..+5 you take 100 percent XP. Below -5 the XP falls fast (80.9, 62.1, 43.0, 23.8 percent, 5.08 percent from -10); at clvl 10 to 12 Blood Moor and Cave Level 1 (mlvl 1 to 2) pay 5 to 43 percent (computed from the table; the review 4.4 item 2a gives the same). Pile size and drop level follow monster level too.
- Leave a zone when it is six or more levels under you and take the next story step (EXP, review 4.4; the SME's one change to look like a real player).
- Area levels (SRC, diablowiki Area Level via Wayback, `mechanics-progression.md` section 5.1; Normal area level, monster types have their own base near it): Blood Moor 1, Cold Plains 2, Stony Field 4, Den of Evil 1, Cave 1/2 level 2, Underground Passage 1 level 4, Dark Wood 5, Black Marsh 6, Tristram 6, Burial Grounds 3, Tower Cellar 1 to 5 level 7, Outer Cloister/Barracks 9, Jail 1 to 3 level 10, Inner Cloister 10, Cathedral and Catacombs 1 to 2 level 11, Catacombs 3 and 4 level 12; Act 2: Sewers 13, Rocky Waste 14, Dry Hills 15, Far Oasis 16, Lost City 17, Tal Rasha's Tomb 17.
- Best area per clvl band (SRC LUG v1.10, max XP per kill, "not iron-clad"): clvl 1 to 11 Normal Act 1, 12 to 18 Act 2, 19 to 23 Act 3, 24 to 31 Act 4, 32 to 36 Act 5, then Nightmare.
- Table of farm zones for the bot at clvl bands (EXP, from the area levels above and the -5 rule): clvl 1 to 5 Blood Moor, Cold Plains, Den; 5 to 9 Stony Field, Dark Wood, Underground Passage, Burial Grounds for Blood Raven at 8 to 10; 9 to 13 Tristram (needs Cain's scroll), Black Marsh, Tower Cellar; 12 to 16 Cathedral, Catacombs, then Andariel; 16 and up Act 2.

## 5. Potion kit, TP scrolls, Akara (kit buy only)

| fact | value | tag |
|---|---|---|
| Minor Healing Potion price | 30 gold (the cheapest item in the game) | SRC, VERIFIED LIVE per `docs/sme/play-economy.md` and `economy-data.md` base `cost` 30 |
| Town Portal Scroll | 100 gold base cost (about 100 in a live shop) | SRC base cost table (`economy-data.md`), price on live shop U |
| Tome of Town Portal | live price 450 at clvl 3 (base 250), `town.tpTomeMinGold` 450 | SRC live per `play-economy.md` |
| Akara | free life, mana and stamina refill; also sells minor/light healing and mana potions | SRC (`class-sorceress.md`, `play-economy.md` ARR vendors page); free heal at Akara/Cain rescue is how humans stay out of debt |
| Kit to carry (human, clvl 8 to 12) | 4 to 6 healing potions, 1 to 2 town portal scrolls; no mana potions on a Paladin before Zeal; go home by waypoint near half life | EXP |
| Boss kit | 8 healing and 2 antidotes for Andariel; Thawing for Duriel (`potion-policy.md`) | EXP |
| Healing amounts | Sorc/Necro/Druid Minor 30, Light 60, Healing 100, Greater 180, Super 320; Amazon/Paladin/Assassin x1.5; Barbarian x2. Minor Mana: 40 Sorc, 30 Paladin | SRC (`play-economy.md`, V POT; per-class table U for 1.14d) |
| Mana potion rule | Paladin: none before Zeal. Sorceress: buy mana potions after healing; drink mana at about 30 percent (EXP target; ours 25 percent) | EXP |
| Belt | Sash/Light belt 8 slots, Belt 12 slots; a belt beats armor at clvl 12 | SRC code `belt.Slots` / EXP |
| Buy order | TP scroll first, then healing for two columns, mana (casters), then a belt; no gambling, no magic vendor gear (a magic Wand cost 1,669 vs 205 normal) | EXP / live per `play-economy.md` |

Rule in one line: buy kit only, pick up every pile, sell every magic and rare, never gamble before about clvl 40 and 100k gold (EXP, `play-economy.md`).

## 6. Gold income, selling and repair (cells mostly EXP; SRC where marked)

| fact | value | tag |
|---|---|---|
| Gold per hour, Normal Act 1, piles | 300 to 1,000 | EXP (`pace-tables.md` 2.3) |
| Gold per hour, selling magic and rare gear, Act 1 | 1k to 3k | EXP |
| Act 1-2 clvl 10 to 15 | piles 1 to 2k/h, selling 3 to 6k/h | EXP |
| Selling cap per item | Normal Act 1 5,000, Act 2 10,000, Act 3 15,000; Nightmare 30,000; Hell 35,000; all vendors pay the same | SRC (`play-economy.md`, VEN) |
| Gold carry limit | 10,000 per character level in the pack; stash 2.5 million | SRC widely quoted, 1.14d offline U |
| Repair | humans repair below 20 to 40 percent durability; Charsi in Act 1, Fara Act 2, Hratli 3, Halbu 4, Larzuk 5; cost grows with item level and missing durability, formula U | SRC / U |
| Death and gold | min(clvl, 20) percent of carried gold lost; the rest drops as a ground pile at the corpse (secondary source, live test still to do); stash is safe | S (`play-economy.md` GOLDSWEEP) |
| Free Rogue | Act 1 Rogue hired for about 100 gold at level 3 (`class-paladin.md` V AS); the Act 2 desert merc gives Prayer (hireling.tsv rows) | SRC (`mercenary.md`, T table) |
| Poverty rule | a human is never poor at clvl 8 to 12 because of three habits: farm where mlvl >= clvl - 5, buy kit only, let Akara heal | EXP (review 4.4 item 2) |

## 7. Paladin mana math (SRC unless marked; skills.txt tables via `class-paladin.md`)

- Mana pool: starting mana 15 (Paladin, T, start value U in the extract), plus 1.5 per clvl and 1.5 per Energy point (SRC TAB). Without Energy: clvl 5 about 21, clvl 10 about 28 (28 mana at clvl 10 with 0 Energy is right, review 4.4 item 3), clvl 12 about 31, clvl 20 about 43. Life: about 55 at clvl 1, +2 per clvl, +3 per Vitality point (TAB). Do not put Energy in before clvl 12 (EXP, the guide takes it then).
- Aura and skill costs at skill level 1 / 10 / 20 (SRC, `class-paladin.md` table TAB):

| skill | clvl | mana | note |
|---|---|---|---|
| Might | 1 | 1 | aura, +40 percent damage at level 1, +10 per level (AB, TAB). Mana cost is the activation (see aura rule below) |
| Sacrifice | 1 | 0 attack mana | +180 percent damage at the guide's skill levels (EXP review); costs 8 percent of damage dealt as life; the damage to self applies only when the hit does physical damage (AB, fetched) |
| Smite | 1 | 2 | shield bash, Charge prerequisite (AB, fetched: mana cost 2) |
| Prayer | 1 | 1 / 2.7 / 4.6 | heals 6+ life per tick at level 1 (S, rate not verified); costs mana (below) |
| Zeal | 12 | 2 | multi-hit: hits = min(2 + level - 1, 5), level 4 is the hit cap; 2 mana at all levels (TAB) |
| Charge | 12 | 9 | needs Smite |
| Blessed Hammer | 18 | 4.75 + 0.25 * slvl (level 1 about 5) | magic damage; BH 1 at clvl 18 in our plan |
| Vengeance | 18 | 4 / 6.2 / 8.8 | needs Zeal |

- Prayer drains mana (AB, fetched, Prayer page): the heal and the cost apply once every two seconds; when any target in range needs healing the mana cost is doubled and the Paladin does not regenerate mana while it heals. When no target needs healing the cost is still applied every two seconds. So Prayer is a rest aura, not a fighting aura (review 4.4 item 3; the cause of the sandbox "out of mana: backing off" lines is UNSURE).
- Veteran early attack pattern: plain Attack under Might and Sacrifice (zero attack mana); Smite is kept as a prerequisite (the `palprofile` card holds the profile data). Zeal at clvl 12, not 6 (`docs/research/class-early-game.md` is wrong on that, and `class-paladin.md` says auras cost nothing; both are on the review's defect list).
- Mana potions on a Paladin are not needed before Zeal (EXP). Energy points: none (EXP, the stat plan holds them back).

## 8. Sorceress mana math

- Mana pool: start about 35, +2 per clvl, +2 per Energy point (SRC TAB, start U); life start about 40, +1 per clvl, +2 per Vitality point. Without Energy: clvl 1 35 mana, clvl 5 43, clvl 10 about 53, clvl 15 about 63, clvl 20 about 73 (computed).
- Early skills (SRC `class-sorceress.md` table): Charged Bolt 3 / 7.5 / 12.5 (skill level 1 / 10 / 20; +0.5 per point is the rule in the review), Ice Bolt 3 flat (4 points never wasted: +15 percent per level to Ice Blast, Glacial Spike, Blizzard, Frozen Orb, Frost Nova), Ice Blast 6 / 10.5 / 15.5 (clvl 6), Frost Nova 9 / 18 / 28 (clvl 6), Warmth +30 percent mana recovery, +12 percent per extra level (TAB), Glacial Spike 10 (clvl 18), Teleport 24 at level 1 then minus 1 per level (clvl 18), Blizzard 23 / 32 / 42 (clvl 24), Frozen Orb 25 (clvl 30).
- The clvl 1 to 8 trap (review 4.4 item 6): a 35 to 49 mana pool is about 6 to 12 Ice Bolts or Charged Bolts; empty at melee range with 40 life kills. Stop leveling in the field and go home to rest rather than drink down the belt (EXP, `class-sorceress.md`).
- Kit: healing first (about 2 columns), then mana potions; Akara sells minor and light mana (SRC). Mana drink at about 30 percent is the veteran target (EXP, `potion-policy.md`).

## 9. Normal-act floors and gear milestones (all EXP unless marked)

| gate | clvl floor | extra |
|---|---|---|
| Den of Evil | about 3 to 5 | skill point and a respec at Akara's quest |
| Blood Raven | 8 to 10 | potions and a ranged or merc support; our bot lost 189 attempts underleveled (OUR) |
| Cathedral and Catacombs | 13 with the merc, 14 without | 4 to 6 healing potions, 1 to 2 TP scrolls |
| Andariel | 14 with the merc, 16 without; speedrun floor 12 ([D2R] DBS, SRC) | 8 healing, 2 antidotes; cold-lightning-fire resist not needed in Normal Act 1 |
| Duriel (Act 2 gate) | 20 to 24 preferred; speedrun checkpoint 17 ([D2R]) | Cold resist or potions; Duriel mlvl 22 and Cold-resist 50 in Normal (`play-act-route.md`) |
| Mephisto / Diablo / Baal (Normal) | 24 / 24+ / 26 to 30 | Ancients quest needs clvl 20 in Normal (S) |
| Nightmare | 30 preferred; Normal Act 5 is good XP to about clvl 40 | resist -40 penalty in Nightmare (SRC XPT section 6) |

Gear milestones (EXP unless marked):
- clvl 1 to 5: starting weapon; hire the Rogue (about 100 gold) when the camp opens (SRC `mercenary.md`).
- clvl 8 to 12: a belt before armor; a shield for Paladin Smite (Smite needs a shield); a cheap normal weapon from Charsi if poor (normal items cost base price; magic items cost 3 to 8 times).
- clvl 12 to 16: Paladin gets Zeal with 4 points over 4 levels; Sorceress Ice Blast from clvl 6.
- clvl 15 to 20: rare or crafted weapon, resists as found; a 3 to 4 socket runeword base waits for runes (Countess runes at clvl 11 to 14).
- Resists in Normal: not required before Act 3 (SRC XPT); the penalty starts in Nightmare.

## 10. Timeline in play-hours (EXP; from `pace-tables.md` section 3, melee/hybrid)

| play-hours | clvl, Paladin or other melee/hybrid | clvl, caster |
|---|---|---|
| 2 | 11 to 14 | 12 to 16 |
| 8 | 24 to 28 | 26 to 30 |
| 24 | 38 to 45 (Nightmare Act 1 to 3) | 40 to 48 |

Normal Act 1 beats run in hours 0 to 4; the table in section 1 is the finer split. XP per hour for humans at clvl 1 to 10 about 15 to 30k and 10 to 15 about 40 to 100k (EXP; our bot measured 13 to 18k/h in the Den, B).

## 11. What this page does not answer (consult d2-sme for these)
- Anything about 1.14d clvl by hour that is measured: no source exists; our own telemetry is the next step (`pace-tables.md` section 6).
- Class builds other than Paladin and Sorceress: see `class-*.md` and `builds-leveling.md`.
- Per-skill damage and breakpoints: `mechanics-combat.md`, `mechanics-breakpoints.md`.
- Whether Prayer or the Smite ranking caused the sandbox mana back-offs: UNSURE (`docs/palprofile.md`).
- Gold per hour and repair formula in 1.14d: U everywhere; calibrate with the bot's logger (`economy-data.md` section 5).

## D2R-only items kept out of this page
Terror Zones, shared stash, runeword ladder rules, and D2R speedrun times (53 minutes for Normal Baal) are D2R; only the DBS speedrun checkpoints are quoted, tagged [D2R].

# D2 SME KB: progression (XP, levels, areas, difficulty, resists), classic LoD 1.14d

Author: SMEKB-A. Tags: **V** fetched reliable source (URL in Sources), **S** secondary/snippet, **U** unverified/memory, **C** computed by us.
Version scope: 1.14d (same rules as 1.13c/1.11 for these systems unless noted). D2R-only items flagged.

## 1. Experience pipeline for one kill (V B15 Amazon Basin Experience page)

Order of operations (each step feeds the next):
1. `Base = MonLvl * MonStats.exp/100` (per monster type; Nightmare/Hell monsters use higher MonLvl).
2. Player-count multiplier: players 1..8 = x1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5 (that is `(n+1)/2`; V A3 Arreat says the same).
3. Monster bonus: Champion/Fanatic/Ghostly/Possessed x3 (+2 mlvl); Berserker, Unique, Super Unique, Minion x5 (+3 mlvl). Bosses only get the x5/+3 if they
   roll random-unique bonuses in NM/Hell, i.e. Griswold, Radament, Nihlathak.
4. Party bonus (see section 3).
5. Hard ceiling 8,388,607 XP per kill (solo only reached at high player counts on some Hell evils, e.g. Diablo at 7 players, Baal 3 players).
6. Level-difference penalty (section 2).
7. Level 70+ penalty (section 2).
8. `% Experience Gained` bonuses applied last (XP shrine +50%, Annihilus small charm 5-10%, Ondal's Wisdom 5%).
No XP for: players, mercs, pets, barricades/doors/towers, Diablo's bone prisons, Baal's festering appendages, Hungry Dead from Blood Raven, Blood Hawks from nests,
sarcophagus mummies, sand maggot young/eggs, Vile Children, Demon Imps from huts, Ice Spawn from Nihlathak. Kills by your merc/pets/returned damage give you XP.
Resurrected monsters give XP only the first time (V A3).

## 2. Level-difference penalties

### 2.1 Character level < 25 (V B15, V A3)
`mlvl - clvl` -> % XP: >=+10: 1.95% (A3 writes 2%); +9 14.8%; +8 35.9%; +7 68.0%; +6 87.9%; -5..+5 100%; -6 80.9%; -7 62.1%; -8 43.0%; -9 23.8%; <-9 5.08%.
(B15 gives the exact /256 fractions: 5/256, 38/256, 92/256, 174/256, 225/256, 256/256, 207/256, 159/256, 110/256, 61/256, 13/256.)
### 2.2 Character level 25..99 (V B15)
Monsters higher than you: XP * clvl/mlvl (A3 states the same). Equal or lower: same lower-side table (100% down to -5, then 80.9/62.1/43.0/23.8, 5.08% beyond -9).
Practical: a monster above your level hurts you proportionally but never gives a 2% floor after clvl 25.
### 2.3 Level 70+ extra penalty (V B15, V A3)
Applied regardless of monster level: clvl 70 95.31%, 71 90.63, 72 85.94, 73 81.25, 74 76.56, 75 71.88, 76 67.19, 77 62.50, 78 57.81, 79 53.13, 80 48.44, 81 43.75, 82 39.06, 83 34.38,
84 29.69, 85 25.00, 86 18.75, 87 14.06, 88 10.55, 89 7.91, 90 5.96, 91 4.49, 92 3.42, 93 2.54, 94 1.95, 95 1.46, 96 1.07, 97 0.78, 98 0.59, 99 0.49 (B15; A3 table identical down to 98).
Consequence: 85+ is extremely slow; level 99 needs ~3.5 billion XP.

## 3. Party XP (V B15, V A3, V B15 notes)
- Players must be in a party and in the same named area ("level"), within 53 1/3 yards (about two screens) of the kill (V B15).
- Bonus = `abs((members-1) * xp * 87)/256` since 1.11 (89/256 in 1.10; Arreat says "35%"): 2 members x1.34, 3 x1.68, 4 x2.02, 5 x2.36, 6 x2.70, 7 x3.04, 8 x3.38 total (V B15 list).
  Overflow bug when ((members-1)*xp*87) >= 2^31 gives negative -> XP set to 1 (V B15).
- Then the pool is split by level ratio: your share = total * yourLevel / sum(levels of members in range) (V B15). Eight equal characters each get ~1/8,
  so a solo player in the same game *without* a party can out-earn a party member per kill (V B15).
- Hirelings take no XP; their kills credit you (V A3). Killing blow takes the XP if neutral players hit the same monster.
- The level-difference penalty (section 2) is applied to each member's own share, which stops low-level members leeching high-level kills (V A3, wording of the Arreat page).
## 4. XP table (total XP to reach level L) (V B15)

Next-level span = difference of consecutive entries. XP bar total at level 99 = 3,520,485,254.

1: 0 | 2: 500 | 3: 1,500 | 4: 3,750 | 5: 7,875 | 6: 14,175 | 7: 22,680 | 8: 32,886 | 9: 44,396 | 10: 57,715
11: 72,144 | 12: 90,180 | 13: 112,725 | 14: 140,906 | 15: 176,132 | 16: 220,165 | 17: 275,207 | 18: 344,008 | 19: 430,010 | 20: 537,513
21: 671,891 | 22: 839,864 | 23: 1,049,830 | 24: 1,312,287 | 25: 1,640,359 | 26: 2,050,449 | 27: 2,563,061 | 28: 3,203,826 | 29: 3,902,260 | 30: 4,663,553
31: 5,493,363 | 32: 6,397,855 | 33: 7,383,752 | 34: 8,458,379 | 35: 9,629,723 | 36: 10,906,488 | 37: 12,298,162 | 38: 13,815,086 | 39: 15,468,534 | 40: 17,270,791
41: 19,235,252 | 42: 21,376,515 | 43: 23,710,491 | 44: 26,254,525 | 45: 29,027,522 | 46: 32,050,088 | 47: 35,344,686 | 48: 38,935,798 | 49: 42,850,109 | 50: 47,116,709
51: 51,767,302 | 52: 56,836,449 | 53: 62,361,819 | 54: 68,384,473 | 55: 74,949,165 | 56: 82,104,680 | 57: 89,904,191 | 58: 98,405,658 | 59: 107,672,256 | 60: 117,772,849
61: 128,782,495 | 62: 140,783,010 | 63: 153,863,570 | 64: 168,121,381 | 65: 183,662,396 | 66: 200,602,101 | 67: 219,066,380 | 68: 239,192,444 | 69: 261,129,853 | 70: 285,041,630
71: 311,105,466 | 72: 339,515,048 | 73: 370,481,492 | 74: 404,234,916 | 75: 441,026,148 | 76: 481,128,591 | 77: 524,840,254 | 78: 572,485,967 | 79: 624,419,793 | 80: 681,027,665
81: 742,730,244 | 82: 809,986,056 | 83: 883,294,891 | 84: 963,201,521 | 85: 1,050,299,747 | 86: 1,145,236,814 | 87: 1,248,718,217 | 88: 1,361,512,946 | 89: 1,484,459,201 | 90: 1,618,470,619
91: 1,764,543,065 | 92: 1,923,762,030 | 93: 2,097,310,703 | 94: 2,286,478,756 | 95: 2,492,671,933 | 96: 2,717,422,497 | 97: 2,962,400,612 | 98: 3,229,426,756 | 99: 3,520,485,254

Handy marks: level 30 = 4.66 M; 40 = 17.3 M; 50 = 47.1 M; 60 = 117.8 M; 70 = 285 M; 80 = 681 M; 90 = 1.62 B (B15).
Level 25->30 needs 3.0 M; 60->65 needs 65.9 M; 85->90 needs 568 M (computed from table).
Death loss (V B15): in NM/Hell, 5%/10% of the XP span of the *current* level (not total XP), never delevels; 75% recoverable from the corpse without save-and-exit. Example B15: death at
level 67 costs 1,006,303 (NM) or 2,012,606 (Hell). Normal difficulty: 0.

## 5. Monster level and item level

- Normal monsters: mlvl = area level in NM/Hell; in Normal each monster type has its own base level (V D3 diablowiki Area Level).
  Champions mlvl+2, uniques/minions +3, super uniques have fixed levels independent of the area (V D3, V D4 gives the monster-level bonus list).
- ilvl of a drop = mlvl of the killer (or chest level; chest levels are area-based, details unclear) (V D4). Items from NPC shops: ilvl = clvl + 5 (cap by act in Normal, see items doc).
- Boss/act-end fixed levels quoted by the diablowiki 1.09 item page: Normal Diablo 40, Nightmare Andariel 45, Normal Baal 60, Nightmare Mephisto 59, highest mlvl Hell Nihlathak 98 and Hell Baal 95 (V D4);
  the Area Level page instead says Hell Baal ilvl 99, Hell Nihlathak 95, Hell Diablo 94 (V D3). **Conflict; both from the same wiki; U which is right for 1.14d.** Treat bosses as ~mlvl 94-99 in Hell.

### 5.1 Area levels (V D3, diablowiki "Area Level" table via Wayback; format Normal/Nightmare/Hell)
In Nightmare/Hell mlvl = the area level below; in Normal, monster types in an area have their own level near the alvl. Some sources differ by 1 for a few areas (Maxroll Act 4 page quotes Chaos Sanctuary-area 27/57 where this table says 28/58/85; S).
Key facts: Hell area levels run 67 (Blood Moor) to 87 (Rocky Summit / Arreat Summit); the 85-level Hell areas (best drops): Pit L1/L2, Mausoleum, Ancient Tunnels, Maggot Lair 3, Sewers L2 (A3), Forgotten Temple/Ruined Fane/Disused Reliquary, Chaos Sanctuary, River of Flame, Worldstone Keep 1-3, Throne, Worldstone Chamber (V D3 table). (C: list built from the table.)

**Act 1** (37 areas): Blood Moor 1/36/67; Cold Plains 2/36/68; Stony Field 4/37/68; Dark Wood 5/38/68; Black Marsh 6/38/69; Tamoe Highland 8/39/69; Den of Evil 1/36/79; The Cave Level 1 2/36/77; Underground Passage Level 1 4/37/69; The Hole Level 1 5/38/80; The Pit Level 1 7/39/85; The Cave Level 2 2/37/78; Underground Passage Level 2 4/38/83; The Hole Level 2 5/39/81; The Pit Level 2 7/40/85; Burial Grounds 3/36/80; Crypt 3/37/83; Mausoleum 3/37/85; Tower Cellar Level 1 7/38/75; Tower Cellar Level 2 7/39/76; Tower Cellar Level 3 7/40/77; Tower Cellar Level 4 7/41/78; Tower Cellar Level 5 7/42/79; Monastery Gate 8/40/70; Outer Cloister 9/40/70; Barracks 9/40/70; Jail Level 1 10/41/71; Jail Level 2 10/41/71; Jail Level 3 10/41/71; Inner Cloister 10/41/72; Cathedral 11/42/72; Catacombs Level 1 11/42/72; Catacombs Level 2 11/42/73; Catacombs Level 3 12/43/73; Catacombs Level 4 12/43/73; Tristram 6/39/76; Secret Cow Level 28/64/81

**Act 2** (27 areas): Rocky Waste 14/43/75; Dry Hills 15/44/76; Far Oasis 16/45/76; Lost City 17/46/77; Valley of Snakes 18/46/77; Canyon of the Magi 16/48/79; Sewers Level 1 13/43/74; Sewers Level 2 13/43/74; Sewers Level 3 14/44/75; Harem Level 2 13/47/78; Palace Cellar Level 1 13/47/78; Palace Cellar Level 2 13/47/78; Palace Cellar Level 3 13/48/78; Stony Tomb Level 1 12/44/78; Halls of the Dead Level 1 12/44/79; Halls of the Dead Level 2 13/45/81; Claw Viper Temple Level 1 14/47/82; Stony Tomb Level 2 12/44/79; Halls of the Dead Level 3 13/45/82; Claw Viper Temple Level 2 14/47/83; Maggot Lair Level 1 17/45/84; Maggot Lair Level 2 17/45/84; Maggot Lair Level 3 17/46/85; Ancient Tunnels 17/46/85; Tal Rasha's Tomb 17/49/80; Tal Rasha's Chamber 17/49/80; Arcane Sanctuary 14/48/79

**Act 3** (27 areas): Spider Forest 21/49/79; Great Marsh 21/50/80; Flayer Jungle 22/50/80; Lower Kurast 22/52/80; Kurast Bazaar 22/52/81; Upper Kurast 23/52/81; Kurast Causeway 24/53/81; Travincal 24/54/82; Spider Cave (Arachnid Lair) 21/50/79; Spider Cavern 21/50/79; Swampy Pit Level 1 21/51/80; Swampy Pit Level 2 21/51/81; Flayer Dungeon Level 1 22/51/81; Flayer Dungeon Level 2 22/51/82; Swampy Pit Level 3 21/51/82; Flayer Dungeon Level 3 22/51/83; Sewers Level 1 23/52/84; Sewers Level 2 24/53/85; Ruined Temple (In Kurast Bazaar) 23/53/84; Disused Fane (In Kurast Bazaar) 23/53/84; Forgotten Reliquary (In Upper Kurast) 23/53/84; Forgotten Temple (In Upper Kurast) 24/54/85; Ruined Fane (In Kurast Causeway) 24/54/85; Disused Reliquary (In Kurast Causeway) 24/54/85; Durance of Hate Level 1 25/55/83; Durance of Hate Level 2 25/55/83; Durance of Hate Level 3 25/55/83

**Act 4** (5 areas): Outer Steppes 26/56/82; Plains of Despair 26/56/83; City of the Damned 27/57/84; River of Flame 27/57/85; Chaos Sanctuary 28/58/85

**Act 5** (23 areas): Bloody Foothills 24/58/80; Frigid Highlands 25/59/81; Arreat Plateau 26/60/81; Crystalized Cavern Level 1 (Crystalline Passage) 29/61/82; Cellar of Pity (Frozen River) 29/61/83; Crystalized Cavern Level 2 (Glacial Trail) 29/61/83; Echo Chamber (Drifter Cavern) 29/61/84; Tundra Wastelands (Frozen Tundra) 27/60/81; Glacial Caves Level 1 (Ancients' Way) 29/62/82; Glacial Caves Level 2 (Icy Cellar) 29/62/83; Rocky Summit (Arreat Summit) 37/68/87; Nihlathak's Temple 32/63/83; Halls of Anguish 33/63/83; Halls of Death's Calling (Halls of Pain) 34/64/84; Halls of Vaught 36/64/84; Hell 1 (Abaddon) 39/60/81; Hell 2 (Pit of Acheron) 39/61/82; Hell 3 (Infernal Pit) 39/62/83; The Worldstone Keep Level 1 39/65/85; The Worldstone Keep Level 2 40/65/85; The Worldstone Keep Level 3 42/66/85; Throne of Destruction 43/66/85; Worldstone Chamber 43/66/85


## 6. Difficulty progression and unlock rules (V A4, V B3)

- Defeat Baal (Act 5) in Normal to open Nightmare, and in Nightmare to open Hell (Classic: Diablo). Hardcore no longer requires beating the game since 1.13 (B3).
- Titles (Expansion): Normal Slayer, NM Champion, Hell Patriarch/Matriarch (hardcore: Destroyer, Conqueror, Guardian) (V A2, V B3).
- XP loss on death: 0 / 5% / 10% (V A2).
- Player/merc resist penalty (Expansion): Normal 0 / Nightmare **-40** / Hell **-100**; Classic -20/-50 (V B3, V A5). Resist is clamped to [-100, cap] (cap 75 default, up to 95) (V A5).
- Scroll of Resistance from Malah's quest: +10 to the four resists (per difficulty); the +10 disappears at death until new game (V A5, V M3).
- Quest gates for the Barbarian Ancients (Act 5): minimum clvl 20 Normal, 40 NM, 60 Hell (S, gamerguides snippet).
- Players level faster in NM at roughly 26-50 and in Hell at 51-75 (V A4). Normal Act 5 is good XP until clvl ~40 (V A4); "Nightmare monsters start at level 36 and quickly rise" (V A4);
  Arreat recommends redoing the Den of Evil quest in Nightmare for its reward when you start the difficulty (V A4).
- Exceptional drops: 10% (20% for special monsters) in NM, 20%/40% in Hell; Elite drops 30%/40% in Hell and start toward the end of NM (V A4).
- Merchants: Exceptional items offered in NM, Elite in Hell (V B3, V A7 vendors). Act 1/2 Hell vendors are not upgraded (V A7).
- NM/Hell monsters: HP, defense, XP, damage, AR and resists scale; monster skill levels +3 in NM (V A4); more immunities.
- Hell gold vendors max sell price 35k (see items doc).

## 7. Safe resist targets and when humans move on (S/U, community heuristics)

Hard facts (V A5, V B3, V M3): displayed resist = gear - penalty; cap 75; 5 max over-cap helps vs Conviction/Lower Resist; magic resist is not penalised.
- **Nightmare entry**: base -40. Typical safe floor before pushing late NM: 0..+30 each of fire/cold/light (Arreat only warns that "lowered resistances will make it more difficult"); aim for 75 in
  fire/cold/light in late NM for Hell prep (S: guides recommend building resist gear during NM, gamerguides snippet). Poison is the one that is often left at 0..50 in NM (U).
- **Hell entry**: needs about 175 raw per element to be at 75 (C). Practical minimums used by players: fire/cold/light >= 50 effective (raw 150) with stronger classes using 75; poison 0..50 effective is common for non-melee (U).
  The Arreat page says in Hell "you'll have a difficult time surviving without a lot of resistances" (V A5).
- **Levels humans use (S/U)**: Normal to ~clvl 40-45; Nightmare to ~clvl 70-75 (one guide: "reach 40-45 on Normal, then 70-75 before Hell", S diablobytes/consolepulse snippet); Hell entry usually clvl 70-80 with a finished build,
  bots/skilled players often enter NM at 28-35 and Hell at 60-70 for specific fast builds (U).
- Other indicators of readiness: chance to hit vs Hell mlvl (see combat doc), FHR/FCR breakpoints, +life, a mercenary on Fanaticism/Defiance aura, and enough potions (belts).
- Classic (non-LoD) differences: penalty -20/-50 (V B3); not applicable to our LoD 1.14d target.

## 8. Bot-relevant summary
1. XP/hour is dominated by (a) mlvl close to or above clvl (before 25 only up to +5, after 25 above-level monsters are scaled by clvl/mlvl), and (b) player count (use /players in offline: x(n+1)/2 XP but x(n+1)/2 HP, i.e. XP per HP is constant, drop quantity changes at odd counts).
2. At clvl 70+ the extra penalty halves XP by 80; leveling beyond 85 is rarely worthwhile.
3. NM/Hell deaths cost 5%/10% of the level span; with a corpse run you recover 75%.
4. Resist penalty makes Hell the first difficulty where unprepared characters die in 2-3 hits from elemental bosses.

## 9. D2R / 1.14d differences
- XP table, penalties, party formula (87/256) and area levels are the same in 1.14d and D2R (S/V B15 states "since 1.11" and D2R notes none).
- D2R adds Terror Zones (not in 1.14d), shared stash and ladder items; Hardcore without Normal completion is 1.13+ (V B3).
- Maxroll pages are D2R; the Basin Experience page cites Phrozen Keep tests for 1.11+.

## Sources
- A3 Arreat Experience https://web.archive.org/web/2016id_/http://classic.battle.net/diablo2exp/basics/experience.shtml (V); A4 Arreat Difficulty .../basics/difficulty.shtml (V); A5 Arreat Resistances .../basics/resistances.shtml (V); A2 Characters .../basics/characters.shtml (V); A7 Vendors .../basics/vendors.shtml (V).
- B15 Amazon Basin Experience https://www.theamazonbasin.com/wiki/index.php?title=Experience (V); B3 Basin Difficulty https://www.theamazonbasin.com/wiki/index.php?title=Difficulty (V).
- D3 Diablo Wiki Area Level https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/Alvl (V, Wayback); D4 Diablo Wiki V1.09 Item Generation https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/V1.09_Item_Generation (V).
- M3 Maxroll damage reductions https://maxroll.gg/d2/resources/damage-reductions (V).
- S snippets: gamerguides difficulty levels https://www.gamerguides.com/diablo-ii-resurrected/guide/characters/builds/difficulty-levels ; diablobytes/consolepulse progression roadmap (search result only, not fetched).

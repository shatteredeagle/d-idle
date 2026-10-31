# D2 SME KB: items, treasure classes, magic find, gold and shops (classic LoD 1.14d)

Author: SMEKB-A. Tags: **V** fetched reliable page (URL in Sources), **S** secondary/snippet/fan, **U** unverified or memory, **C** computed by us.
Version: LoD 1.14d. The Basin per-item drop tables carry the note "drop chance has not been updated for 1.13" (V B16), so rune/unique odds below
may be slightly off for 1.14d; treat as order-of-magnitude. D2R differences are flagged.

## 1. Item quality order and what MF touches

Pipeline when a monster dies or a chest opens (V D4 diablowiki 1.09 item generation page; the principles hold in 1.14d, numbers changed in 1.10):
1. **NoDrop roll**: depends on monster type and player count. MF has no effect.
2. **Treasure class (TC) walk**: every monster has a max TC; nested TC picks (weights); item type picked. MF has no effect. Class-specific drop weights: Assassin claws 2/3 of
   normal, other class-only items and rods (wands/staves/scepters) 1/3 (V D4).
3. **Quality roll** in this order, first success wins: Unique > Set > Rare > Magic > Superior > Normal > Low quality (V D4). Magic Find acts only here.
4. Then ethereal (about 5%, never on sets) and sockets (normal/superior only, about 1/3) (V D4).
- Bosses always drop at least a magic item (unless the drop is a rune/gem); super uniques drop 4 potions and 2 items, random bosses 1 item (V D4).
- A rolled unique/set with no matching unique/set for that base (or monster level too low) falls back: failed Unique becomes a **Rare with 3x durability**, failed Set a **Magic with
  2x durability** (V D4; matches common play knowledge "3x durability rare = failed unique"). Same fallback if the unique already dropped once this game; "one of each unique per game" is
  stated by Blizzard but D4 notes reports of duplicates (V D4 caveat).
- Jewels check only Rare/Magic; charms are always Magic; runes, keys, potions, scrolls do not roll quality (V D4).
- Max 6 drops per monster (only act bosses reach it; gold piles may exceed visually). Chests are not capped (V D4).

## 2. Magic Find

### 2.1 What MF is and who gets credit (S diablobytes D5, S almars A11)
- "Better chance of getting magic items" changes quality only; no effect on quantity, base type, runes, gems, jewel/charm stats, sockets, gambling, cube crafting, super chests, gold (S D5: also says runes are in the "Good" TCs, MF has zero effect).
- MF is read when the killing blow lands (weapon-swap trick: swap to an MF set before the last hit); a merc killing blow adds merc MF to yours before the curve (S D5, S A11). No hard cap.
- Sources of MF (1.14d examples, S): Harlequin Crest +50 (Shako), Skullder's Ire 1.25/clvl, Enigma 1/clvl, Wealth +100, Perfect Topaz +24 in helm, Chance Guards, War Traveler, Goldwrap, Ali Baba. Ladder/D2R-only items excluded.

### 2.2 Diminishing returns (S D5, V D4 for the older table)
`EffectiveMF = MF * Factor / (MF + Factor)`; Factor = 250 (Unique), 500 (Set), 600 (Rare), infinite (Magic = full MF) (S D5; consistent with the table below).

| Raw MF | Magic | Rare | Set | Unique | Unique x vs 0 MF |
|---|---|---|---|---|---|
| 50 | 50 | 46 | 45 | 42 | 1.42 |
| 100 | 100 | 86 | 83 | 71 | 1.71 |
| 200 | 200 | 150 | 143 | 111 | 2.11 |
| 300 | 300 | 200 | 188 | 136 | 2.36 |
| 400 | 400 | 240 | 222 | 154 | 2.54 |
| 500 | 500 | 273 | 250 | 167 | 2.67 |
| 750 | 750 | 333 | 300 | 188 | 2.88 |
| 1000 | 1000 | 375 | 333 | 200 | 3.00 |

(S D5.) The older 1.09-era table on the diablowiki page (V D4) used different factors: 250% MF = 250/220/215/193 for magic/rare/set/unique, 500% = 500/340/322/253,
1000% = 1000/460/421/295: i.e. the 1.10+ curve is the one above (S). Community rule of thumb: first ~250% MF matters most; beyond ~400% it slows kill speed more than it helps
(S D5 worked example: 300 MF at 10 s/run loses to 200 MF at 7 s/run by ~27% uniques/min).
Per-kill benefit: a boss with 100 MF finds uniques about 1.7x as often, 300 MF about 2.4x (S D5, derived).
**Formula (U, from memory of itemratio.txt, partly consistent with V D4 numbers):** for each of unique/set/rare/magic:
`chance = (Base - (ilvl-qlvl)/Divisor) * 128;  chance = chance*100/(100+EffMF);  chance -= chance*TCmod/1024;  success if rnd(chance) < 128` with Base/Divisor
Unique 400/1, Set 160/2, Rare 100/2, Magic 34/3; `TCmod` is the TC's Unique/Set/Rare/Magic column (e.g. 983 for act bosses: 1 - 983/1024 => ~25x better than a base monster).
Cross-check: V D4's act-boss unique chance at 100% MF for a qlvl-1 item is 12.15%; the formula gives 12.2% when ilvl-qlvl is about 48 (C; plausible, not proven).

### 2.3 Actual quality odds (V D4, from the 1.09 page, qlvl-1 item; "400% MF" column; shows structure, not exact 1.14d numbers)
| Killer class | Unique (100/400/800% MF) | Set | Rare | Magic |
|---|---|---|---|---|
| Act boss / Cow King | 12.2 / 20.4 / 23.4% | 28.1 / 49.5 / 60.6% | 19.1 / 19.3 / 13.5% | 40.6 / 10.8 / 2.5% |
| Countess | 4.0 / 6.8 / 7.7 | 10.2 / 19.1 / 24.0 | 21.7 / 37.5 / 45.3 | 64.2 / 36.6 / 23.0 |
| Super uniques, random bosses | 2.2 / 3.7 / 4.3 | 5.7 / 10.9 / 13.8 | 23.2 / 43.2 / 54.3 | 68.8 / 42.2 / 27.6 |
| Champion | 2.2 / 3.7 / 4.3 | 5.7 / 10.9 / 13.8 | 5.4 / 10.0 / 12.6 | 86.7 / 75.3 / 69.4 |
| Normal monster | 0.48 / 0.81 / 0.93 | 1.27 / 2.46 / 3.11 | 1.25 / 2.48 / 3.21 | 38.8 / 94.2 / 92.8 (rest normal) |

Quality odds drop with item qlvl: act boss at 400% MF, qlvl 1: 20.4% unique/49.5% set; qlvl 45: 18.8/45.4; qlvl 89: 17.1/41.1 (V D4). Elite uniques are therefore harder than low-level ones.

## 3. Treasure classes (TC)

- Structure (V D4, V D6 diablowiki Treasure Classes): 30 weapon TCs and 30 armor TCs, "Weap/Armo 3, 6, 9 ... 90" in steps of 3; a TC holds the base items whose qlvl falls in its band (e.g.
  TC 12 = qlvl 10-12). Runes live in 17 rune TCs, 2 runes in each of the first 16 and Zod alone in the 17th. Jewelry, charms, gems, potions, keys sit in per-act "Good"/"Junk" TCs. 
- A monster has a list of possible TCs; the game rolls from the *highest* down, and the highest are very unlikely, e.g. Hell Baal reaches Armor 87 in about 0.001% of drops (V D4).
- TC quality columns (Basin TC listings, V B16): Act bosses "Unique 983, Set 983, Rare 983, Magic 1024"; Countess 883/883/883/1024; Council 650/800/800/1024; Hephasto/Blood Raven 800/800/900-972/1024;
  Nihlathak/Radament/Summoner 900/900/900(972)/1024; normal super uniques 800/800/972/1024; Duriel 983/983/983/1024 (V B16). Lower number = better (value out of 1024 is the fraction removed). (S interpretation)
- Example TC listing (V B16, Normal Mephisto): "Picks: 7; total weight 80; NoDrop 15 (18.75%), gold 5, Act 4 Equip A 52, Act 4 Junk 5, Act 4 Good 3"; Countess uses a special pair "Countess Item" / "Countess Rune" with no NoDrop and Picks -2 (V B16).
- Base-item levels: sample TC 87 weapons/armor (qlvl 85-ish): Archon Staff, Colossus Blade, Giant Thresher, Sacred Armor, Thunder Maul, Berserker Axe, etc. Unique/set items are only available from a TC whose base list contains the base and when monster level >= the unique/set level (V D6, V D4). 
  TC 87 uniques include Tyrael's Might (qlvl 87), Mang Song's Lesson (86), Death Cleaver/Executioner's Justice etc. (V D6).
- Champions use "Act N Champ" TCs; unique/minion packs "Act N Unique"; "Act N Super" for super uniques (V B16 item listings).

### 3.1 No-drop and player count (V D4, S D5)
- NoDrop values: normal monsters 100/160 = 62.5% at 1 player; most chests 100/140 = 71.4%; act-end bosses (Diablo, Baal) 15/80 = 18.75%; champions, super uniques, random bosses 0% (V D4).
- Player-count term (V D4): 1 for the killer + 1 per partied player in the same area + 0.5 per partied player in another area + 0.5 per unpartied player in the game. In single player/TCP `/players X` each simulated extra player counts 0.5
  (S D5: /players 3 = 2 NDE). Drops increase at odd player counts only (S A11, S D5); even counts only raise monster life/XP.
  NoDrop for a regular monster: 1 player 62.5%, /players 3 39.1%, 5 24.4%, 7 15.3% (S D5). Boss packs are unaffected. Countess runes are better at low player counts (special table) (S D5).
- HP scales with players: HP*(n+1)/2 (V A3).

## 4. Drop rates by monster and rune (V B17 Amazon Basin item pages, baseline /players 1, 0 MF, not updated for 1.13)
Chance "1 in X" for one specific item; Hell values:

| Rune | Countess (H) | Andariel (H) | Duriel (H) | Mephisto/Baal (H) | Hephasto (H) | Nihlathak (H) | Council (H) | Act 5 super (H) | Act 5 unique pack (H) | Act 5 champ (H) |
|---|---|---|---|---|---|---|---|---|---|---|
| Ko | 160 | 10,816 | 10,818 | 12,918 | 13,564 | 13,790 | 20,723 | 26,280 | 52,560 | 113,034 |
| Fal | 212 | 14,321 | 14,324 | 17,105 | 17,961 | 18,260 | 27,440 | 34,799 | 69,597 | 149,672 |
| Lem | 318 | 21,482 | 21,486 | 25,658 | 26,941 | 27,390 | 41,159 | 52,198 | 104,396 | 224,509 |
| Pul | 423 | 28,544 | 28,549 | 34,092 | 35,797 | 36,394 | 54,689 | 69,357 | 138,714 | 298,309 |
| Um | 635 | 42,816 | 42,824 | 51,138 | 53,696 | 54,591 | 82,034 | 104,035 | 208,071 | 447,464 |
| Mal | 739 | 49,865 | 49,874 | 59,558 | 62,536 | 63,579 | 95,541 | 121,164 | 242,328 | 521,137 |
| Ist | 1,109 | 74,798 | 74,812 | 89,337 | 93,805 | 95,368 | 143,311 | 181,746 | 363,493 | 781,706 |
| Gul | 123,870 | 87,178 | 87,194 | 104,123 | 109,330 | 111,152 | 167,030 | 211,827 | 423,655 | 911,086 |
| Vex | 185,805 | 130,766 | 130,791 | 156,184 | 163,996 | 166,729 | 250,546 | 317,741 | 635,483 | 1,366,630 |
| Ohm | 216,650 | 152,474 | 152,503 | 182,112 | 191,220 | 194,407 | 292,138 | 370,488 | 740,976 | 1,593,498 |
| Lo | 324,975 | 228,711 | 228,754 | 273,168 | 286,830 | 291,610 | 438,207 | 555,732 | 1,111,465 | 2,390,247 |
| Sur | - | - | 266,791 | 318,590 | 334,523 | 340,099 | 511,071 | 648,139 | 1,296,277 | 2,787,693 |
| Ber | - | - | 400,186 | 477,884 | 501,785 | 510,148 | 766,606 | 972,208 | 1,944,416 | 4,181,540 |
| Jah | - | - | - | 557,426 | 585,304 | 595,059 | 894,204 | 1,134,027 | 2,268,053 | 4,877,535 |
| Cham | - | - | - | 836,139 | 877,956 | 892,589 | 1,341,306 | 1,701,040 | 3,402,080 | 7,316,302 |
| Zod | - | - | - | Baal 2,926,220 | 3,072,531 | 3,123,740 | - | 5,953,029 | 11,906,059 | 25,604,428 |
(Mephisto and Baal values equal to within 1; Zod only from Baal/Hephasto onward as Hell TCs allow it.) Rune sums: Gul-and-up sell for up to 35,000 each. The runeword uses of Ber: Beast, Chains of Honor, Destruction, Enigma, Eternity, Infinity, Last Wish, Wrath (V B17).
Takeaways: Hell Countess is by far the best Ko-through-Ist source (1 in 160 to 1,109 per run); Gul and higher come from bosses; MF does not apply to runes (S D5). Countess drops use "Countess Rune" TC with up to Ist/Gul (V B16 counts; S for exact cap).

Gold TCs: act bosses and champions have a `"gld,mul=N"` entry with N=1280 in Normal, 1536 in Nightmare, 2048 in Hell (V B18 Basin gold TC page): that is a gold multiplier of N/256 = 5x/6x/8x (S, interpretation).

## 5. Item level, affix level and magic items

- ilvl of a drop = monster level (area level in NM/Hell; champion +2, unique +3). Items bought from NPCs: ilvl = clvl + 5; caps in Normal by act: A1 12, A2 20, A3 28, A4 36, A5 45; no cap in NM/Hell (V D4).
- **Affix level** (V D4):
  ```
  ilvl = min(ilvl, 99); if qlvl > ilvl: ilvl = qlvl (temp)
  if magic_lvl > 0: alvl = ilvl + magic_lvl
  else alvl = (ilvl < 99 - qlvl/2) ? ilvl - qlvl/2 : 2*ilvl - 99     (integer math, cap 99)
  ```
  magic_lvl: Circlet +3, Coronet +8, Tiara +13, Diadem +18 (diadems are always alvl 99); normal/exceptional wands, all staves and orbs +1 (V D4). Arreat says affix chosen with `qlvl <= ilvl+2` (V A10) and uses ilvl (monster level) directly; the datamined alvl formula above is the better description; U which applies on exceptional bases without testing.
- Magic item composition (V A10): prefix+suffix 25%, prefix only 25%, suffix only 50%; at most one prefix and one suffix.
- Affixes are picked almost flat across levels (a level-50 monster is about as likely to give a lvl-48 affix as a lvl-14 one), unlike TCs (V D4).
- Cube recipe "3 perfect gems + 1 magic item" rerolls with ilvl 100 (V D4); mods can come from chests only for some (V D4: a +45 life grand charm needs chest ilvl).
- Sockets (V D4): normal/superior drops ~1/3 socketed; max sockets by ilvl with breakpoints at mlvl 25 and 40; Larzuk reward gives max sockets for normal/superior, 1-2 on magic, 1 on crafted/rare/set/unique, none if already socketed or non-socketable. Magic suffixes give 1-4 sockets; rares and crafted at most 1 ("of Mechanics").

## 6. Uniques, sets and jewelry odds (V D4)
- A monster must have mlvl >= the unique's or set's qlvl and its TC must contain the base; the first monster that can drop Goldwrap is Diablo because the heavy belt (qlvl 20) drops early but Goldwrap needs qlvl 36 (V D4).
- Ring/amulet unique rarity table (V D4, rarity values): Nagelring ilvl 10 rarity 15; Manald Heal 20/15; Stone of Jordan 39/1 (about 1.9% of unique rings at best, 1/31 at Normal Diablo mlvl 40 or NM Andariel 45; 1/51 at mlvl 59-60); Bul-Kathos' Wedding Band 66 only late Nightmare/Hell and 1/52 at best.
  Sets: Cathan's Seal ilvl 15, Angelic Halo 17 (rings); Civerb's Icon 13, Cathan's Sigil 15, Angelic Wings 17, Vidala's Snare 19, Arcanna's Sign 20, Iratha's Collar 21, Tal Rasha's Adjudication 26, Tancred's Weird 27, Telling of Beads 39 (amulets).
  Farm tip derived: SoJ best from mlvl 39-52 sources (V D4).

## 7. NPC shops (V A7 vendors, V D4)
- Vendor inventory depends on act, difficulty and character level; NPC stock items use ilvl = clvl+5 (see above); chance of Exceptional/Elite versions: `Exceptional = (floor(100*(ilvl-qlvl)/2)+1)/100 %`, Elite `(floor(100*(ilvl-qlvl)/4)+1)/100 %`, with `ilvl = clvl - 5 + rnd(10)` (V D4);
  best case at clvl 99: NM exceptional 10.7%, Hell exceptional 18.3%, Hell elite 2.7% (V D4). Arreat also says Exceptional items start appearing in late Normal vendors, Elite in Hell, and Act 1/2 Hell vendors are not upgraded except throwing weapons (V A7); diablowiki: "you never see Elite items in Act 1 or 2 due to a bug" (V D4).
- **Stock refresh** (V A7): all players left town; or an NPC not traded with for the previous 4 minutes if players remain (diablowiki says 5 minutes and "completely motionless", V D4; conflict, assume 4-5 min); Arreat also says the inventory is reset when you leave town but not the act, with an example that contradicts itself (Rogue Encampment to Bloody Foothills no reset, to Blood Moor reset; Bloody Foothills is an Act 5 area, so the example looks like a typo) (V A7; U on the exact rule). Gamble stock regenerates each time the Gamble window opens (V A7).
- **Max sell price per item** (V A7, V D2): Normal Act 1 5,000; Act 2 10,000; Act 3 15,000; Act 4 20,000; Act 5 25,000; Nightmare 30,000; Hell 35,000 (all acts). Save a valuable item for the next act/difficulty if it caps out (V A7).
  All vendors pay the same; repairing before selling does not raise the price (V A7). Most valuable per inventory cell: wands with charges up to 35k in two cells; any exceptional/elite body armor 35k in six; flawless gems 7,500 each (skulls double) (V D2).
- Faster sell: Ctrl+left-click sells without confirmation (V A7).
- Identify: Cain identifies everything for free if you rescued him, otherwise 100 gold per item; only magic, rare, set, unique need ID; scrolls/tomes ID one item per charge (V D7 diablowiki Scrolls and books).
- Scrolls/tomes: a tome holds up to 20 scrolls of its own type; bought empty or found with 1-5 charges; tomes usually sell with >= 2; tome price/resell varies with charges (V D7, V PD1 planetdiablo misc page). **Exact buy/sell prices (U, from memory of 1.14d vendor lists): Scroll of Town Portal 100, Scroll of Identify 80, Tome of Town Portal 450, Tome of Identify around 1,000-1,300, key price not remembered, none verified; no fetched source gives numbers and PD1 shows "?"; a bot should read prices from the vendor window.**
- Keys stack 12; chests consume one per open (V D8 diablowiki Misc).
- Repair cost and base price formula: not found in a fetched source (U). Repair does not change sell value (V A7).

## 8. Gambling (V A8 Arreat, V D4)
- One NPC per act sells gamble stock; items unidentified and revealed on purchase; price rises with clvl (V A8). MF does not affect gambling (V A8, S D5).
- Stock items are -5..+4 levels around your clvl, min 5 (V A8); an item type appears once your clvl reaches qlvl-4 (V A8; the ilvl roll is `clvl - 5 + rnd(10)` min 5 max 99, V D4). Item categories unlock with clvl; at 48 all appear (V A8).
- Quality odds per Arreat (V A8, **plausibly outdated**): Unique 1/2000 (0.05%), Set 1/1000 (0.10%), Rare 1/10, Magic rest. 
- Exceptional upgrade `max(0, (ilvl - qlvl_exc)*0.9% + 1%)`, Elite `max(0, (ilvl - qlvl_elite)*0.33% + 1%)` (V A8); diablowiki gives `(100*(ilvl-qlvl)/2+1)/100 %` and elite `... /4 ... * (1-pExc)` (V D4; the two disagree on constants, treat as U).
- Rare/set/unique gambling is the way to buy rings and amulets, especially in Hell (V A8). Opening and closing the window rerolls the stock rapidly (V A8), and buying all items on one screen is no better than one at a time (V A8).
- Known bugs (V D4): fewer boots than expected at high clvl; Elite belts and elite Bardiche/Great Sword cannot be gambled (belts upgrade to boots codes). Affixes use alvl from the `clvl-5+rnd(10)` ilvl; to always have +2 skills (alvl 90) on amulets you need clvl 94 (V D4).
- Gambling is the main gold sink for high-gold characters; typical bot approach (S): gamble rings/amulets/circlets at high clvl; U for EV.

## 9. Gold
- Carry cap: 10,000 per clvl (V D2): level 1 10k, level 15 150k, level 99 990k. Stash cap: 50k (L1-9), 100k (10-19), 150k (20-29), 200k (30), 800k (31), then +50k every two levels to 2.5M at 98-99 (V D2).
  Gold needs no inventory space; spending uses pocket gold first, then stash (V D2).
- Death: lose clvl% (max 20%) of total (pocket+stash) gold in multiplayer; single player never touches the stash and exempts 500*clvl (V A2, V D2).
- Gold Find (GF%) multiplies the amount dropped by monsters and chests; does not affect vendor prices (V D2). Wealth +300 GF, etc.; high GF characters easily exceed 500-1000% (V D2).
- In parties, gold picked up is split evenly among party members on that level; those at cap are skipped (V D2).
- Ground gold disappears after about 10-15 min (V D2).
- Dropped gold per monster and the exact formula: U. Basin TC: gold multiplier 5x/6x/8x for act bosses/champions by difficulty (see section 4, S).

## 10. 1.14d vs D2R
- Same pipeline; D2R and 1.14d share MF curves in the sources we found (S D5 is D2R-framed but claims the same code). D2R-only: Terror Zones, ladder-only runewords/items, larger stash, shared tab, auto-gold pickup (U). RotW: new item types (Colossal jewels, Warlock items) per Basin tags (B1 = Amazon Basin wiki pages in general, https://www.theamazonbasin.com/wiki/).
- Basin item pages mark "D2R only"/"LoD Ladder" flags: runewords like Enigma (Jah+Ith+Ber), Infinity (Ber+Mal+Ber+Ist) are in 1.10+ LoD; "Last Wish" etc. were ladder in LoD, non-ladder in D2R; "Infinity Spear" form added in D2R 2.4 (V B17).
- `players X` command is single-player/TCP only; nothing changes in 1.14d offline.

## Sources
- A7 Arreat vendors https://web.archive.org/web/2016id_/http://classic.battle.net/diablo2exp/basics/vendors.shtml ; A8 gambling .../basics/gambling.shtml ; A10 magic items .../items/magic.shtml ; A3 experience; A2 characters; D2 gold (below) (V).
- D2 Diablo Wiki Gold https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/Gold ; D4 V1.09 Item Generation https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/V1.09_Item_Generation (V); D6 Treasure Class https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/Treasure_Class (V); D7 Scrolls and books https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/Scrolls_and_books (V); D8 Misc .../Misc (V); D5 Diablobytes MF guide https://www.diablobytes.com/guides/magic-find/ (S, D2R-framed, March 2026); A11 Almar's MF https://almarsguides.com/Computer/Games/Diablo2/Farming/MagicFind/ (S).
- B16/B17/B18 Amazon Basin item/monster pages: https://www.theamazonbasin.com/wiki/index.php?title=Ber_Rune (and Ko/Fal/.../Zod_Rune), Mephisto, Countess, Gold (V; "drop chance not updated for 1.13").
- PD1 PlanetDiablo misc items https://gamespy-archives.quaddicted.com/sites/www.planetdiablo.com/diablo2/items/misc/index.html (V, prices missing).

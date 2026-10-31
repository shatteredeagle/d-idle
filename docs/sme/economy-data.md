# Economy data: vendor prices, stock, gold, gambling (classic 1.14d, Normal Act 1)

Written 2026-10-29 by SMEKB-D. Fills the gaps in `play-economy.md` (SMEKB-C). Target: classic LoD 1.14d offline, not D2R.
Tags: **V** = fetched this session (source below) or read in our own docs/live logs, **S** = snippet or secondary, **U** = memory/inference.
`[D2R]` = D2R-only or 1.14d status unknown.

Sources (all fetched by SMEKB-D unless marked):
- D2DATA = blizzhackers/d2data JSON dump of the game tables (`misc.json`, `armor.json`, `weapons.json`, `monstats.json`, `levels.json`), https://raw.githubusercontent.com/blizzhackers/d2data/master/json/ (V as game data; the exact patch is U, but the base `cost` values are unchanged between 1.10 and 1.14d as far as I know, U). Prices below called "base cost" are the `cost` column.
- VEN = Arreat Summit mirror, vendors: https://classic.hiveworkshop.com/diablo2exp/basics/vendors.shtml (V)
- GAM = Arreat Summit mirror, gambling: https://classic.hiveworkshop.com/diablo2exp/basics/gambling.shtml (V)
- IGT = Diablo Wiki "Item Generation Tutorial" (Wayback snapshot): https://web.archive.org/web/2023/https://diablo2.diablowiki.net/Item_Generation_Tutorial (V; chapter "Item Generation at Shops")
- GLD = Diablo Wiki "Gold": https://web.archive.org/web/2023/https://diablo2.diablowiki.net/Gold (V)
- GF = Diablo Wiki "Gold Find": https://web.archive.org/web/2023/https://diablo2.diablowiki.net/Gold_Find (V)
- GMB = Diablo Wiki "Gambling": https://web.archive.org/web/2023/https://diablo2.diablowiki.net/Gambling (V)
- DUR = Diablo Wiki "Durability" (ethereal selling): https://web.archive.org/web/2023/https://diablo2.diablowiki.net/Durability (V)
- OUR = our docs (`docs/items.md`, `docs/leveler.md`, `docs/town.md`) with live numbers (V where they say VERIFIED LIVE)

Direct curl of diablowiki.net and purediablo.com returns Cloudflare blocks (also from the worker host browser); the Wayback copies work from curl with a browser User-Agent.

## 1. Headline findings

1. Vendor BUY price of a plain normal item equals the base `cost` in the tables for ordinary armor/weapons (V live: a normal Wand cost 205 = `wnd` cost 205, OUR leveler.md; D2DATA `wnd` cost 205). Magic items from a vendor cost several times that (live: magic Wand 1,669 vs 205 normal, V OUR).
2. The base `cost` of the things a bot buys: TP scroll 100, ID scroll 80, minor healing 30, minor mana 60, Tome of TP 250 (live 450 at clvl 3, V OUR, so the tome price includes charges, U how), Tome of ID 200.
3. Akara (Act 1, Normal) sells ONLY the minor potions (healing 30, mana 60), stamina, antidote, thawing, the two scroll types, the two tomes, keys and staves/wands/scepters/daggers. Light/normal potions come from Act 2+ (Lysander, Drognan, Ormus, Malah, Jamella). A Normal Act 1 character never has anything better than Minor Healing to buy (V D2DATA vendor Min/Max columns, details in section 3).
4. Rings (50,000) and amulets (63,000) cost a fixed amount to gamble in `misc.txt` (V D2DATA `gamble cost`; GMB agrees: "Rings are always 50,000 and amulets always 63,000"). That is out of reach for clvl 8-14. Gambling weapons/armor at Gheed costs 3,000-20,000 for base items of qlvl 1-12 (V D2DATA `gamble cost`). Early gambling is not an income source; skip it until the first 20k+ is banked.
5. No source I could fetch gives a verified gold-per-hour figure for clvl 8-14. Gold piles in Normal Act 1 are small (tens of gold); the real income is selling magic/rare gear and wands. Per-area numbers in section 5 are U estimates and are labeled so. The honest conclusion is: measure it with the bot (calibration logger, `docs/items.md` P9).

## 2. Vendor prices (buy), classic base costs

All numbers are base `cost` from D2DATA unless noted; the vendor charges this for a normal-quality item. Quality multiplies it (magic and above cost much more, section 4).

### Potions, scrolls, tomes, keys (misc.json, V)
| Item | Code | Base cost | Notes |
|---|---|---|---|
| Minor Healing Potion | hp1 | 30 | live 30 confirmed (V OUR leveler.md "cheapest potion") |
| Light Healing | hp2 | 75 | Act 2+ vendors only |
| Healing | hp3 | 125 | Act 3+ (Ormus, Malah, Jamella) |
| Greater / Super Healing | hp4 / hp5 | 250 / 500 | not sold in Normal Act 1-2 |
| Minor / Light / Normal Mana | mp1 / mp2 / mp3 | 60 / 150 / 300 | mp2 from Act 2, mp3 from Act 3 |
| Rejuvenation / Full Rejuvenation | rvs / rvl | 400 / 1500 | `spawnable 0`: never sold, drop only |
| Stamina | vps | 25 | |
| Antidote | yps | 40 | |
| Thawing | wms | 25 | |
| Scroll of Town Portal | tsc | 100 | |
| Scroll of Identify | isc | 80 | |
| Tome of Town Portal | tbk | 250 (live 450, V OUR) | max stack 20, spawn stack 5 (D2DATA); U whether price = 250 + per-charge share. Live: 450 at clvl 3. |
| Tome of Identify | ibk | 200 | same stack fields |
| Key | key | 45 | Gheed and Akara sell them (max 9 and 2 per stock) |
| Arrows / Bolts | aqv / cqv | 256 (a stack) | Charsi/Gheed; stack price U, that value is the cost field for a full quiver |

Scroll price: 100 gold per TP scroll is the base cost; live scroll price on our client is U (OUR town.md says price equals base, docs mark it U). Identify scrolls are 80 each.

### Armor, cheap Act 1 bases (armor.json, V; qlvl = `level` column)
| Item | qlvl | Cost | Sold by (Normal A1 columns) |
|---|---|---|---|
| Cap | 1 | 64 | Charsi, Gheed |
| Quilted Armor | 1 | 140 | Charsi, Gheed |
| Buckler | 1 | 68 | Charsi, Gheed |
| Leather Gloves / Boots / Sash | 3 | 80 / 80 / 64 | Charsi, Gheed |
| Leather Armor | 3 | 481 | Charsi, Gheed |
| Skull Cap / Small Shield | 5 | 441 / 410 | Charsi, Gheed |
| Hard Leather Armor | 5 | 1,060 | Charsi, Gheed |
| Studded Leather | 8 | 2,385 | Charsi, Gheed |
| Large Shield | 11 | 1,214 | Charsi, Gheed (Fara too) |
| Ring Mail | 11 | 4,428 | Charsi |
| Light Belt (qlvl 7) / Belt (qlvl 12) | 7 / 12 | 192 / 495 | Charsi/Gheed + Fara |

Belt capacity: sash 8 slots, light belt 8, belt 12 (our layout.go, V). A Sash costs 64 and is the cheapest 8-slot belt.

### Weapons, staves, wands (weapons.json, V)
| Item | qlvl | Cost | Sold by |
|---|---|---|---|
| Club / Short Sword / Katar | 1 | 32 / 72 / 72 | Charsi, Gheed (Gheed has the club; Charsi the others) |
| Short Staff | 1 | 168 | Akara (max 9) |
| Short Bow | 1 | 100 | Charsi |
| Javelin | 1 | 5 | Charsi, Gheed |
| Wand | 2 | 205 | Akara (max 6) |
| Dagger | 3 | 60 | Akara (5), Drognan (Act 2) |
| Hand Axe | 3 | 170 | Charsi, Gheed |
| Scepter | 3 | 350 | Akara |
| Spiked Club | 4 | 225 | Charsi, Gheed |
| Spear / Scimitar / Hunter's Bow | 5 | 300 / 274 / 350 | Charsi, Gheed |
| Long Staff | 8 | 463 | Akara (3) |
| Mace / Saber | 8 | 463 / 466 | Charsi |
| Gnarled Staff / Yew Wand | 12 | 766 / 745 | Akara (Yew Wand 2) |

Vendor ilvl: every vendor item has `ilvl = clvl + 5`, capped in Normal at 12 in Act 1 (V IGT). So a clvl 8-14 character sees ilvl 12 maximum at Charsi/Gheed/Akara. Normal-quality stock is limited to qlvl <= 24. Items spawn magic once ilvl >= 25 (not reachable in Act 1 Normal), so Act 1 vendors sell NORMAL items (plus the `MagicLvl`-gated magic ones: Akara/Charsi `MagicLvl` is 255 for most items, so most Act 1 stock is never magic, V D2DATA + IGT note 2). The 1,669-gold magic wand seen live is an exception, so check by reading the price: normal wand = 205.

## 3. Act 1 vendor stock (what exists, V D2DATA `Charsi/Gheed/Akara Max` columns)

| Vendor | Role | Normal-quality stock (Max > 0) |
|---|---|---|
| **Akara** (Act 1) | potions, scrolls, tomes, caster weapons | minor HP (1), minor MP (1), stamina (4), antidote (4), thawing (2), TP scroll (17), ID scroll (17), TP tome (2), ID tome (2), key (2), Short Staff (9), Wand (6), Dagger (5), Scepter (4), Long Staff (3), Dirk (3), Gnarled Staff/Yew Wand etc. at higher qlvl |
| **Charsi** (Act 1) | weapons, armor, repair, repair (Larzuk sockets are Act 5; Charsi imbue is a quest reward) | arrows (5), bolts (4), javelins (3), throwing knives (2), katar (2), club-type and sword/axe/bow bases, Cap, Quilted, Buckler, Leather, Hard Leather, Studded, Skull Cap, Small/Large Shield, Ring Mail at qlvl <= ilvl |
| **Gheed** (Act 1) | gambling, keys, bases | key (9), arrows (3), bolts (4), club, short sword, katar, axes, spears, bows and the same body-armor/helm/shield bases as Charsi |
| **Fara** (Act 2) | armor, repair | Light Belt, Heavy Gloves/Boots, Belt, Helm, Large Shield, Ring Mail; she is where humans buy the first belt (S play-economy.md, D2R guide) |

Notes:
- Each "Max" is the maximum quantity of that item type in one stock refresh, not a guarantee (V IGT: the `*Min` field is ignored by the game). A given Akara refresh may have 3 wands or none.
- Stock refreshes: all players left town, or an NPC not traded with for 4 minutes; leaving the town area (not just the act) regenerates all vendors (V VEN). Gambling stock regenerates every time you click Gamble (V VEN, GAM). A bot that re-opens the trade window gets new gamble stock; the trade stock stays.
- Nightmare/Hell upgrade the stock (V IGT) but Hell Act 1-2 vendors are not upgraded except throwing weapons (V VEN).

## 4. Buy vs sell, selling caps, repair

### What is verified
- **All vendors pay the same for an item** ("you do not have to wheel and deal", V VEN).
- **Sell cap per item:** Normal A1 5,000, A2 10,000, A3 15,000, A4 20,000, A5 25,000; NM 30,000; Hell 35,000 (V VEN, GLD). A clvl 8-14 character in Act 1 Normal can never get more than 5,000 for one item; hold anything that hits the cap until Act 2+ (V VEN).
- **Repairing before selling does not raise the sale price** (V VEN).
- **Ethereal items sell for 25% or less** of normal price (V DUR "cutting it by 75% or more"), and are not shown for buy-back (V DUR).
- **Most profitable per inventory cell:** wands (often 35,000 at the cap with 2 cells, late game) and +charges scepters/orbs/claws; body armor and helms are the most expensive by raw value (V GLD). In Act 1 Normal values are far below the cap. Set/unique items sell for "far less" than a magic item of similar mods because their sale value is a fixed number in the tables (V GLD).
- **Death gold rule (single player):** no stash loss, and 500 gold per clvl is exempt; multiplayer loses clvl % up to 20% of carried plus stash (V GLD). So at clvl 10, 5,000 gold carried is safe on death.

### What is NOT verified (no source fetched; U)
- Exact sell = f(base cost, quality, affixes) formula. Community lore: vendors pay roughly a quarter of the buy price for plain items, e.g. a TP scroll sells for 25 (U, memory; not confirmed by anything I could fetch).
- Repair cost formula: cost grows with missing durability and item level/quality; no verified 1.14d numbers (Japanese D2R wiki says "no conclusion in English", S, https://wikiwiki.jp/diablo2r/Repair%20cost%20data). Throwing weapons repair for 1 gold per stack refill (V DUR "repaired for 1 gold").
- Identify scroll sell price, tome sell price: U.
- **Recommended method:** have the bot log, for each sell and repair, `(code, quality, ilvl, affix count, gold)`; fit a table (docs/items.md P9 "calibration logger"). The formula has a fixed shape in the item tables (`price` modifiers `divide/multiply/add` on every affix, V IGT affix field description) so a handful of samples per quality pins it down.

## 5. Gold income, Normal Act 1, clvl 8-14

### What is solid
- Gold piles scale with monster/area level; a Normal Act 1 monster has level 1-7 (V D2DATA `levels.json`/`monstats.json`): Blood Moor 1, Cold Plains 2, Stony Field 4, Dark Wood 5, Black Marsh 6, Burial Grounds 3, Den of Evil 1, Cave 1 / Cave 2 2, Tristram 6, Tower Cellar 1 and Forgotten Tower 7, Blood Raven 10, Andariel 12.
- Gold find multiplies the pile: "100% GF doubles it" (V GF). Gold Find gear/Lem is not available at clvl 8-14.
- Gold TC entries are `gld` (base) or `gld,mul=N` with N a multiplier whose unit is U (256 or 1024; the Lurker snippet says 256): Act 1 "Citem" TCs use 1280, Andariel (N) 1536, (H) 2048 (V mofr/Diablerie TreasureClassEx.txt, a clone's data copy of the table); IGT: "gold with a 2048 multiplier means decent sized stacks". Plain monsters use the base `gld`.
- A formula posted on the Lurker Lounge (2003, "Hammerman", seen only as a search snippet, S): base pile = rand up to 5 x ilvl, plus ilvl, then the TC multiplier / 256, then gold find. Another snippet (S) puts a typical Normal monster at roughly 1-5 gold per monster level. I could not fetch the thread or confirm in 1.14d (U).

### Estimates (U): derive per hour from the formula above and kill rates
Using "1-5 gold per mlvl per drop" and "gold drops on about 10-20% of kills" (TC no-drop weights, U):
| Area (Normal) | mlvl | Gold per pile (U) | Piles per kill (U) | Gold/hour at ~600-900 kills/h (U) |
|---|---|---|---|---|
| Blood Moor / Den | 1 | 2-7 | 0.15 | 200-900 |
| Cold Plains / Cave 1-2 | 2 | 3-12 | 0.15 | 300-1,500 |
| Stony Field / Burial Grounds | 3-4 | 5-20 | 0.15 | 450-2,700 |
| Dark Wood / Black Marsh / Tristram | 5-6 | 6-35 | 0.15 | 550-4,700 |
These are rough bounds, 10x uncertain. Our own live logs support the low end: 0-3 gold in 29 min of Blood Moor (bot skipping piles, V docs/g1-log.md attempt 5, cited by play-economy.md). The conclusion that survives: **gold piles are worth well under 1,000 gold per hour at clvl 8-14 in Act 1, and selling gear is 5-10 times that.** Replace this table with measured values from the bot.

### What sells best from Act 1 drops (S/U from V caps and GLD)
1. **Magic and rare weapons/armor** (with affixes) beat any normal item many times over; a rare circlet/amulet/ring is worth the most per cell (S, play-economy.md sources). Sell magic items; for normal items check price per cell, because many white items sell for under 10 gold (quilted armor base 140 buy, likely U ~35 sell).
2. **Wands, scepters, staves with +skills or charges**: the highest value per cell; a common early drop (S GLD).
3. **Class-specific items** (necro heads, druid pelts, sorc orbs, amazon spears, barb helms) sell well if magic (S PDW via play-economy.md).
4. **Throwing knives, javelins, fulminating potions** are worth hoarding in D2R Normal Act 1 per beginner guides (S, gamepressure/Maxroll snippets, D2R).
5. **Typical price of a magic Act 1 item** (U, no source fetched): 100-600 gold for a lightly modded base. Our live data point: a magic Wand listed at 1,669 for sale at the vendor; vendor sells at about 8x the plain base. If vendor markup relative to sale is x4, a magic wand would sell for about 400, which is U.
Do not sell: runes, gems the Cube needs, Cain/quest items, charms and good rings/amulets (see play-economy.md).

## 6. Gheed gambling economics (early)

- **Item level at the gambling table:** gamble stock is clvl -5..+4 (min ilvl 5); the type list opens as clvl rises (V GAM).
- **Prices:** rings 50,000 and amulets 63,000 flat (V GMB, D2DATA); others rise with clvl; base `gamble cost` at clvl 1-12 for armor/weapons is 3,016-31,397, e.g. Cap 3,016, Wand 3,910, Hand Axe 4,510, Leather Armor 4,360, Broad Axe 18,172, Crystal Sword 31,397 (V D2DATA, the displayed price at your clvl is U). Cost per roll is therefore higher than any Act 1 income at clvl 8-14.
- **Odds** (V GAM): Magic ~89.85%, Rare 10%, Set 0.1%, Unique 0.05%; Exceptional/Elite chance pV = max(0, (ilvl - qlvl_ex) * 0.9% + 1%), pE = max(0, (ilvl - qlvl_el) * 0.33% + 1%); the game will not give exceptional/elite below clvl 25. MF does not affect gambling (V GAM).
- **Verdict for a bot:** do not gamble in Act 1 Normal. The expected value per roll of a base item at 3-10k is a 90% magic item worth a few hundred gold, so EV is deeply negative until gold is plentiful (Act 2+ and Nightmare, or Hell with gold find). Keep `skip gambling` as in `docs/items.md` first-purchase list.
- **Stock mechanics:** re-clicking Gamble refreshes the table; buying all items on a page does not change odds (V GAM). Gheed's Fortune charm lowers prices (V GMB) but is not Act 1 relevant.

## 7. Practical shopping order and budget for clvl 8-14 (derived, V prices + U income)
1. TP scrolls (100 each) until you can afford the tome (450 live). Tome of TP after clvl ~6.
2. Minor healing x8-12 (30 each = about 300), minor mana for casters (60 each).
3. A Sash (64) or Light Belt (192): cheapest capacity upgrade.
4. Cap/Quilted/Buckler basics (about 270 total) if the body slot is empty; skip Leather Armor (481) and Studded Leather (2,385) unless you are rich.
5. Hire the Rogue at 8-9 (price shown in the hire dialog; our clvl 13 reading was 360, V OUR items.md).
6. Everything else waits for Act 2 income. Never buy vendor magic items (8x markup).

## 8. Open questions (need live calibration)
- Sell price per quality/ilvl (log it). Repair price per durability point.
- Whether the tome price is 250 + n x per-charge (n live) or a quality/vendor markup.
- Real gold pile size per area (log pickups with the area name and monster level).
- Actual Gheed gamble price at clvl 8-14 for rings and amulets (fixed 50k/63k per the tables; confirm in game).
- Akara higher potion tiers in Act 1 Normal: tables say none; confirm in the vendor window at clvl 14.

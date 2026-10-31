# Items, economy and runewords for the offline bot (agent ITEMRESEARCH)

Target: classic LoD **1.14d**, offline single player (SP). No D2R facts unless flagged "D2R".
Tags: **V** verified (our code, or the game tables of this install read by `internal/data` from `/tmp/d2mpq` on the worker host, Patch_D2 over d2exp over d2data, read 2026-10-29 with a throwaway dump, nothing committed; or fetched page, URL given), **S** secondary or snippet, **U** unverified / memory.
No game was run. Table rows quoted below are copied from that dump (items, runes.txt, cubemain.txt, belts).

## 0. What exists today (V, read from the tree)

| area | where | what it does | gap |
|---|---|---|---|
| pickit | `internal/loot/pickit_starter.json`, `rules.go`, `loot.go` | ordered rules, first match wins; actions keep/stash/sell/drop, tiers S..C, `needId` for unidentified jewelry; stat conditions need an identified item | see section 2 |
| sell value | `loot.SellValue` | `cost/4 * qualityFactor` (U, "UNVERIFIED approximation" per code) | never calibrated against a real vendor |
| valuables | `loot/valuables.go` | stash before a dangerous trip: stash verdicts, value >= `ValuableMinGold` 1500, or rule prio >= 70 | fine |
| identify | `internal/town/identify.go` | Identify scrolls or tome, right-click tool then left-click item; Cain NOT used (dialog unmapped); whole flow UNVERIFIED live | no rule decides what is worth identifying beyond `needId` jewelry |
| cube | `internal/town/cube.go` `Transmute(inputs)` | puts named base codes in the Cube, presses Transmute, returns result codes; V live 2026-10-29 (act2live cubemix) | no planner that decides which recipe to run |
| gear scoring | `internal/leveler/score.go` `ChooseGear` | one number per item: base defence/damage plus stat list weights (life 0.6, mana 0.3, res 1.2 each, FCR 1.5, all skills 40, class skill 30, tab skill 25, MF 0.15 ...), hidden-affix guess when unidentified (magic 8, rare 20, set 25, unique 30, crafted 18), superior x1.15, ethereal x1.25 | no runeword awareness, no breakpoints, no per-class stat priority, ethereal bonus is questionable (can't repair eth, see 4.4) |
| `internal/gear` | `gear.go`, `ammo.go` | weapon/ammo model, builds; NOT item rating | n/a |
| belt | `internal/town/belt/plan.json` | clvl 1-12 columns hp,hp,mp,hp, max tier 2; 13-30 tier 4; 31-60 `hp,hp,mp,rv` tier 5; 61+ `hp,rv,mp,rv` | see section 6 |
| merc | `internal/town/merc.go` | hire flow V live (Palli clvl 13: row cost 360, one click hires); revive flow documented in `docs/progression/merc.md`, not coded | revive budget policy missing |
| runeword data | `data.DB.Runewords`, `RunewordAvailableOffline` | reads runes.txt; treats the 23 ladder-flagged rows as NOT offline | **probably wrong, see 3.1** |
| old runeword plan | `docs/plan/data/runewords.json` | rune lists correct (checked against runes.txt below), thin | superseded by section 5 |

## 1. Keep / sell / ignore, item class by class

"Value" = `cost` field of the base (V table), vendor sells for roughly a fraction of it (U, calibrate, package P9).

### 1.1 Gold, potions, scrolls, keys (V pickit, policy S)
- Gold: pick up everything above `GoldMinStack` (10). Gold is the only currency; it pays potions, repair, merc revive, gambling.
- Potions: pick healing/mana only while the belt plan wants them (`needed: potion`), always keep Rejuvenation (rvs 400 / rvl 1500 base value, V items table). Super potions beat buying: sell-neutral, take them.
- TP/ID scrolls: only while the tome is not full (`needed: scroll`). Tomes: one of each (`tbk` base 250, `ibk` base 200). Real purchase price of the first TP tome was 450 gold at clvl 3 (V live, `docs/gear.md` AMALIVE notes), so vendor price is not the base cost.
- Skeleton keys (`key`, 45): useless offline (no trade); keep only for chests if KeyTarget > 0 (already `needed: key`).

### 1.2 Runes (S tier, always stash)
- Rune table (V, `r01..r33`): El 11, Eld 11, Tir 13, Nef 13, Eth 15, Ith 15, Tal 17, Ral 19, Ort 21, Thul 23, Amn 25, Sol 27, Shael 29, Dol 31, Hel **0** (the table has Hel's level requirement 0, cost 1715: a known quirk, V in table), Io 35, Lum 37, Ko 39, Fal 41, Lem 43, Pul 45, Um 47, Mal 49, Ist 51, Gul 53, Vex 55, Ohm 57, Lo 59, Sur 61, Ber 63, Jah 65, Cham 67, Zod 69. Level requirement of a runeword = highest rune (+ the item's own).
- Rule: stash every rune. Rune stash is the bot's main long-term currency (cube upgrades, runewords). Never sell, never drop.
- Early value ranking for what to actually spend: Tal, Eth, Ral, Ort, Thul, Amn, Sol (Stealth, AP, Spirit, Insight, Lore, section 5); keep one stock of El/Eld/Tir/Nef/Eth/Ith only as cube feedstock (3 -> 1).

### 1.3 Gems (V tables, cube V)
- Cost field: chipped 500, flawed 1500, standard 5000, flawless 15000, perfect 30000 (V, amethyst row; skulls 1000 chipped ... 100000 perfect). Cube recipes (V cubemain): 3 of one type and grade -> next grade, no rune needed, same for skulls.
- Keep rule: keep and stash gems of grade flawed and above (the pickit already stashes flawless+perfect, topaz-standard and skulls). Chipped: sell unless the cube planner has a plan (3 chipped -> flawed costs only inventory cells; value per cell is low). Practical policy: keep chipped/flawed only up to 2 each type (so the third completes a triple), sell the rest. Not coded.
- What gems are actually for: rune upgrades above Thul (section 4.2), Larzuk-free socket recipes (perfect gem + two runes, 4.5), crafted recipes (perfect gem + jewel + rune, 4.6), and filling sockets (section 5 weapon/armor choices: ruby/topaz/diamond in a socketed base).
- Socket fillers while no runeword exists: Perfect/flawless gem in a helm (ruby life leech ... fine), shields (sapphire/diamond), weapons (ruby). U, not worth coding before runewords.

### 1.4 Jewels, charms, jewelry
- Jewel (`jew`, base 1000): keep and stash (input of the crafting recipes, 4.6; also socket filler). Pickit: yes. Rule extension: identify and keep if "of Fervor"/life-type later (U).
- Charms: base costs scha 2000, mcha 1000, lcha 600 (V). Unique charms (Annihilus, Hellfire Torch, Gheed's) already S. Magic/rare charms: keep only if one of life >= 20 (stat 7, `shift 8`), all resists (stat 39/41/43/45 >= 5 on a small charm), +skill charms (class tab) or FHR; the pickit currently stashes ALL magic/rare charms "A" with no stat filter, which will fill the stash (1 stash = 10x10 cells in classic? U). Needs identify first (charms are magic/rare, stats hidden until ID).
- Rings/amulets: `good jewelry` rule keeps identified ones with life 20+, FCR 10+, all skills, or any of the four resists >= 15 (anyStats). Rare/magic unidentified jewelry is picked and identified (`needId`) - good. Add: a ring/amulet that adds +MF is useless for the bot (MF weight 0.15 in score.go), drop MF-only. Extra rule: any ring with **FHR+FCR+life** and **Stone of Jordan/Nagelring/Raven Frost** by unique index (needs unique table, P6).
- Crafted jewelry from the cube: see 4.6.

### 1.5 Rares, sets, uniques, magic, plain, by base
Pickit now: unique and set -> stash (A, prio 85), magic/rare/crafted -> sell (B, prio 5), low/normal/superior -> drop, except `socketed bases`.
- **Uniques and sets**: still stash (the unique chance is rare, cheap to carry), but only after identification and only if (a) it is an upgrade by the gear scorer or (b) the base is in the good-base list (below), (c) else sell. Unique/set quality is readable from the item unit before ID (`Item.Quality`), but which unique it is needs identification. Useful uniques of the bot's likely ranges (all U, from memory of the item list, check `data.UniqueByName`):
  - Value-based: any unique/set body armor, helm, belt, boots, gloves up to clvl +5 of the bot is probably worth wearing because uniques roll 1-2 fixed useful stats; let the scorer decide (unknown-hidden bonus 30/25 in score.go already).
  - Named stash-always (U): Stone of Jordan, Bul-Kathos' Wedding Band, Nagelring, Manald Heal, Raven Frost, Dwarf Star, Highlord's Wrath, Mara's Kaleidoscope, Metalgrid; Harlequin Crest (Shako), Andariel's Visage, Vampire Gaze, Guillaume's Face, Tarnhelm, Skin of the Vipermagi, Arkaine's Valor (Normal-Nightmare body), Skullder's Ire, Tyrael's Might (Hell), Magefist, Frostburn, Trang-Oul's pieces, Goblin Toe, Gore Rider, War Traveler, Waterwalk, Arachnid Mesh, Verdungo's Hearty Cord, String of Ears, Shaftstop, Stormshield, Spirit Ward, Hellslayer... Not exhaustive. **Pickit cannot do names today**: `d2.Item.FileIndex` (unique index, offset +0x28) is documented "UNVERIFIED" in `internal/d2/items.go`. Package P6.
- **Rares**: sell all, except rare jewelry/circlets/belts/boots/gloves with good stats once identified (scorer). Identify every rare ring/amulet/circlet only (ID scroll 80 base). Rare weapons/armor: sell unidentified (score unknown bonus 20 vs 0 is not worth an ID scroll below ilvl ~?; U).
- **Magic**: sell, except jewelry (`jewelry to id`), charms (stat filter), and magic items with 3+ sockets on good bases? (socketed magic items cannot become runewords: runewords need white sockets, V Arreat Summit "Rune Word Rules": no magic/rare/set/unique, https://web.archive.org/web/2012/http://classic.battle.net/diablo2exp/items/runewords.shtml).
- **Normal (white) and superior bases**: this is where runeword bases come from. Keep (stash) these bases when found, tier A:
  - exactly the max-socket bases below, because Larzuk (Act 5, Siege on Harrogath reward) gives a plain base the maximum sockets its base allows (S: Maxroll D2R socket page, https://maxroll.gg/d2/items/sockets, "Base items receive their maximum possible sockets" depending on ilvl; magic 1-2; rare/set/unique/crafted 1; difficulty does not change it; the quest reward is per difficulty so three socketings per character; 1.14d identical, U). A base whose max equals the runeword's rune count has no variance.
  - already socketed white items with the exact count a target runeword needs (2, 3 or 4). Current rule `socketed bases` stashes "4+ sockets" only, so it MISSES the 2- and 3-socket bases for Stealth, Lore, Rhyme, AP, Leaf, Smoke and WILL stash 5-6 socket bases that fit nothing early except Insight/Fortitude-with-more-runes.
  - Max sockets of the bases (V `gemsockets`): **body armor 2**: Quilted, Leather, Hard Leather, Studded Leather, Scale, Chain, Splint, Plate, Field Plate, Ghost/Serpentskin/Demonhide/Trellised, exceptional equivalents; **3**: Ring Mail, Breast Plate, Light Plate; **4**: Gothic Plate, Full Plate Mail, Ancient Armor. **Shields 3**: Large, Kite, Tower, Gothic (and elite Scutum..), **paladin-only 4**: Targe, Rondache, Heraldic, Aerin, Crown, Monarch? (Monarch `uit` is 4, req clvl 54 per table), Aegis, Ward. **Swords**: Crystal Sword 6, **Broad Sword 4**, Long Sword 4, Claymore/Giant/Bastard 4, Flamberge 5, Great Sword 6, Short/Scimitar/Sabre/Falchion 2, War Sword 3. **Staffs**: Short 2, Long 3, Gnarled 4, Battle 4, War Staff 6. **Polearms**: Bardiche 3, Voulge 4, Scythe 5, Poleaxe 5, Halberd 6, War Scythe 6. **Helms 2**: Cap, Skull Cap, Helm, Full Helm, Bone Helm; **3**: Great Helm, Crown, Mask.
  - Practical bases: Spirit sword = Broad Sword / Long Sword (4, str 48/55); Spirit shield = Monarch (Paladin shield for Paladin: Heraldic etc.); Insight = Voulge/Bill (4); Leaf = Short Staff/Jo Staff (exactly 2 sockets); Stealth = Quilted/Leather/Hard Leather/Studded Leather (2); AP = Large/Kite/the NAS/Gothic (3); Lore = any 2-socket helm.
  - Ethereal bases: ethereal weapons/armor can be socketed and made into runewords (S) but cannot be repaired normally, durability is lower; avoid for the bot (it will never repair them, `town.repair` presses Repair All; U whether Repair All skips ethereal).
- **Low quality** (crude/cracked/damaged/low): drop. There is a recipe to normalise them (rune + chipped gem), not worth it.

## 2. Pickit gaps (concrete edits, data only unless noted)
1. Socketed bases: add rules for exact 2, 3, 4 sockets on `tors|shld|helm|weap`-typed white/superior items (stat 194 is the socket-count stat the existing rule reads) instead of "4+ only". Prio 40.
2. Unsocketed white bases from the list in 1.5 (Broad Sword, Long Sword, Voulge, Short Staff, Quilted/Leather/Hard Leather/Studded, Large/Kite/the NAS/Gothic shields, 2-socket helms): stash at clvl <= 30 while Larzuk is still ahead, tier B otherwise. `codes` glob supports this.
3. Magic/rare charms: add a stat condition (life >= 20, resists, skills) instead of "all".
4. Gems: add "keep chipped/flawed up to N" via a `needed: gem`-style hook; otherwise cube-upgrade chains never fire. Today sell rule (`low gems`) throws chipped/flawed gems away. Package P2.
5. Unique/set: split into "worth ID" vs "sell". Needs `FileIndex` verified (P6).
6. Supply rules `needed: potion` already tie to belt plan; keep.
7. `jewelry to id`: also magic/rare circlets (`circ` is in the list), fine.

## 3. Runeword rules of the game (1.14d)

### 3.1 The 1.14d table (V, runes.txt via `internal/data`, 78 complete rows) and the ladder trap
- Rune order, sockets and allowed bases come from the rows below (V). Socket count = number of runes; **white (normal or superior) item with exactly that many sockets**, correct type, no magic/rare/set/unique (S, https://web.archive.org/web/2012/http://classic.battle.net/diablo2exp/items/runewords.shtml).
- **Ladder trap (V table, U meaning)**: 23 rows carry the "server" flag that `data.Runeword.Ladder` reads, including **Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Widowmaker, Winter, Wrath** (V list). Likewise cube recipes **Dol -> Hel and every rune upgrade above** (`r14..r32` rows) and the elite-unique upgrade recipes have `ladder=true` (V cubemain). `RunewordAvailableOffline` returns false for all of these and so would steer the planner away from Spirit/Insight. My belief (U, strong, community knowledge): offline SP makes everything, ladder-flagged included, because the flag only blocks Battle.net non-ladder characters. Evidence for the recipes: Arreat Summit cube page states the Hel-and-above rune upgrades "will only work for Single-Player, Open, or Ladder Characters, not for Normal Characters on the Realms" (S, https://web.archive.org/web/2012/http://classic.battle.net/diablo2exp/items/cube.shtml). For the runewords I have no fetched source. **Treat as open question Q1 and verify live before the planner hard-depends on it (package P10); change `RunewordAvailableOffline` to ignore `Ladder` once confirmed.**
- Runeword item flag in memory: `d2.FlagRuneword = 0x04000000` (V layout in `internal/d2/items.go`), `FlagSocketed 0x800`.
- Inserting runes: pick the rune with the cursor and left-click it onto the socketed item in the inventory (or right-click flows); order matters, one rune at a time, a wrong order bricks the base (U live; no code today). Not a cube operation.

### 3.2 Larzuk and the socket sources (see also 1.5)
- Larzuk (Act 5, quest "Siege on Harrogath"): reward adds sockets; see 1.5 for numbers (S, Maxroll D2R). Do it on the best white base you hold when the quest is done in that difficulty. Normal Larzuk is the first chance: pick the Spirit sword or shield, not Stealth armor, if only one (stronger and Larzuk is also available Nightmare/Hell).
- Cube socket recipes exist in 1.14d (V cubemain, copied): `Tal + Thul + Perfect Topaz + normal body armor`, `Ral + Amn + Perfect Amethyst + normal weapon`, `Ral + Thul + Perfect Sapphire + normal helm`, `Tal + Amn + Perfect Ruby + normal shield`; random sockets 1..max (S Arreat cube page). Perfect gems are expensive; use only for 2-socket max bases where success rate is high (armor 2, helm 2).
- Rare + 3 Perfect Skulls + Stone of Jordan adds 1 socket (V cubemain); not worth it.
- Shops: vendors sometimes stock socketed white items (U) but not reliably.

## 4. Cube recipes that pay (V = `cubemain.txt` rows of this install, `enabled=true`)

### 4.1 Always on, cheap
- `3 El -> Eld -> Tir -> Nef -> Eth -> Ith -> Tal -> Ral -> Ort -> Thul`: 3 runes of the same name -> next (r01..r09, V). No gem.
- `3 chipped/flawed/standard/flawless gem of one type -> next grade` (V). Skulls too.
- `3 Health Potions (any grade) + 3 Mana Potions + 1 chipped gem (gem0) -> Rejuvenation` (V), `+ standard gem (gem2) -> Full Rejuvenation`, `3 Rejuv -> 1 Full Rejuv`. Base values: rvs 400, rvl 1500, healing potions 30-500 (V). Use only on spare drops of potions the belt does not need; do not buy potions for this.
- `Ort + weapon -> repaired`, `Ral + armor -> repaired` (non-ethereal) - free repair, costs a rune; not worth runes above Ral. Likewise `Hel + TP scroll + socketed item` unsockets and DESTROYS the gems (V). Avoid.
- `1 axe + 1 dagger -> throwing axe`, `1 spear + arrows -> javelins`, `arrows <-> bolts` (V): Amazon/ammo helpers only.
- `3 Perfect Gems + 1 magic item -> new random magic item` and `6 perfect skulls + rare` rerolls: way too expensive early (perfect gem 30000 base). Ignore.
- `3 magic rings -> 1 magic amulet`, `3 amulets -> ring`: ilvl 75% clvl, junk; low value, skip.

### 4.2 Rune ladder with gems (V cubemain, quoted)
`3 Thul + Chipped Topaz -> Amn`, `3 Amn + Chipped Amethyst -> Sol`, `3 Sol + Chipped Sapphire -> Shael`, `3 Shael + Chipped Ruby -> Dol`, `3 Dol + Chipped Emerald -> Hel`, `3 Hel + Chipped Diamond -> Io`, `3 Io + Flawed Topaz -> Lum`, `3 Lum + Flawed Amethyst -> Ko`, `3 Ko + Flawed Sapphire -> Fal`, `3 Fal + Flawed Ruby -> Lem`, `3 Lem + Flawed Emerald -> Pul`, `2 Pul + Flawed Diamond -> Um`, `2 Um + Standard Topaz -> Mal`, `2 Mal + Standard Amethyst -> Ist`, `2 Ist + Standard Sapphire -> Gul`, `2 Gul + Standard Ruby -> Vex`, `2 Vex + Standard Emerald -> Ohm`, `2 Ohm + Standard Diamond -> Lo`, `2 Lo + Flawless Topaz -> Sur`, `2 Sur + Flawless Amethyst -> Ber`, `2 Ber + Flawless Sapphire -> Jah`, `2 Jah + Flawless Ruby -> Cham`, `2 Cham + Flawless Emerald -> Zod`. Ladder flag on everything from Dol -> Hel up (V, see Q1).
- Cost reasoning (V item cost fields): a Thul (8960 base) is worth 3 Ort (6860); upgrading by cube is cheaper than buying nothing - runes are not sold by vendors, so the cube is the only way to turn surplus low runes into the ones Spirit/Insight need. The bot should convert only when the target runeword lacks that exact rune, never "up" surplus blindly: from Tal up, 3-for-1 costs 3 drops.
- Practical early targets and their cube feedstock (Amn needs 3 Thul + a chipped topaz; Thul needs 3 Ort...). In Tal-equivalents: 3 Tal = 1 Ral, 3 Ral = 1 Ort, 3 Ort = 1 Thul, so one Thul = 27 Tal. Do not expect to cube a Thul for Spirit; farm Countess/Hell for them (section 7).

### 4.3 Socketing and normalising recipes (V)
See 3.2. Also `El + chipped gem + low armor -> normal armor`, `Eld + chipped gem + low weapon -> normal weapon` (V cubemain shows r01/r02 rows with `ilvl=1` per S): makes a white base from a low-quality one; ilvl drops to 1 (kills Larzuk socket count, S). Do not use.

### 4.4 Ethereal repair note
`Ort`/`Ral` repair recipes require non-ethereal items (V `noe`).

### 4.5 Upgrading uniques / rares (V)
`Ral + Sol + Perfect Emerald + normal unique weapon -> exceptional` (V r08 + r12 + gpg). `Tal + Shael + Perfect Diamond + normal unique armor -> exceptional` (r07 + r13 + gpw). Elite versions (`Lum + Pul`, `Ko + Lem`...) are ladder-flagged. Useful only for a good unique whose base is outgrown; skip at first.

### 4.6 Crafted items (V rows, quality "crafted"; mid-game pay)
Recipe shape: `magic item (specific base) + Jewel + Rune + Perfect gem -> crafted item of that base with a guaranteed affix group`. Hit-power (sapphire), Blood (ruby), Caster (amethyst), Safety (emerald). Rune/base pairs from the table (rune number = standard ladder, e.g. r04 Nef, r05 Eth, r06 Ith, r07 Tal, r08 Ral, r09 Ort, r10 Thul, r11 Amn, r12 Sol):
- Caster Amulet: magic amulet + Jewel + **Ral** + Perfect Amethyst (cheapest strong one, 30000 base gem + Ral). Caster Ring: Jewel + Amn + Perfect Amethyst.
- Caster Belt: Light Belt (`vbl`) + Jewel + **Ith** + Perfect Amethyst. Caster Boots: `lbt` Boots + Jewel + Thul + PAm. Caster Gloves: Leather Gloves (`lgl`) + Jewel + Ort + PAm. Caster Helm: Mask + Jewel + Nef + PAm. Caster Body: Light Plate + Jewel + Tal + PAm. Caster Shield: Small Shield + Jewel + Eth + PAm. Caster Weapon: Rod (`rod`) + Jewel + Tir + PAm.
- Hit-power weapon: magic blunt + Jewel + **Tir** + Perfect Sapphire (`blun`); blood weapon: magic axe + Jewel + Ort + PRuby; safety: magic Spear + Jewel + Sol + PEm etc. Full list is the `CU|` rows (V). Crafted items get +1-3 random affixes from a group by ilvl (U); the bot cannot control quality.
- These all need a Perfect gem (30000 base) = 27 chipped gems; only viable once a perfect gem drops; keep a perfect gem rule (already S).

## 5. Runewords worth making: by stage and class (rune lists V from runes.txt; effects V from table properties)

Format: name (runes in socket order, sockets = rune count, allowed base types, key V stats). clvl req = highest rune (V above) + base (Hel is 0).
Early-game access = Countess (Tal/Eth/Ral/Ort etc. in Normal/NM, U), Normal Cows, Hell bosses, shopping gambles, Larzuk. Aim only for runewords whose runes the bot already holds.

### Stage A, Normal clvl 15-30 (cheap: El..Amn)
| runeword | runes | base (V types) | V effects | use |
|---|---|---|---|---|
| **Stealth** | Tal Eth | `tors` (2 sockets) | FCR 25, FRW 25, FHR 25 (`balance2`), +6 dex, +15 stam, 3 phys magic res | Everyone. First armor runeword; Tal clvl 17 |
| **Leaf** | Tir Ral | `staf` (2 sockets) | +3 fire skills tab, Firebolt/Warmth/Inferno +3, cold res 33, def/lvl | **Sorceress fire** (pre-Spirit); not other classes |
| **Ancient's Pledge** | Ral Ort Tal | `shld` (3 sockets) | res all 13, cold res 30, +50% defence, damage to mana 10 | Everyone; shield 3-os bases (Large/Kite/the NAS/Gothic) |
| **Rhyme** | Shael Eth | `shld` 2 | res all 25, block 20 + FBR 20, no freeze, +25 MF, +50 gold | Everyone with a shield; Shael clvl 29 |
| **Steel** | Tir El | `swor,axe,mace` 2 | IAS 25, +20 ED, +3 min/max dmg, open wounds 50 | cheapest melee weapon for Paladin/Barb/Ass (not claws) |
| **Malice** | Ith El Eth | `mele` 3 | +33 ED, open wounds 100, **monster cannot heal**, -100 enemy defence, -5 life regen on you, -1 light | skip (regen penalty) |
| **Strength** | Amn Tir | `mele` 2 | +35 ED, +20 str, +10 vit, crushing 25 | melee, mid-NM |
| **Nadir** | Nef Tir | `helm` 2 | +50% def, +10 def, -3 light radius, +5 str | cheap Normal helm |
| **Zephyr** | Ort Eth | `miss` 2 | Amazon bows | cheapest Amazon bow runeword |

### Stage B, clvl 25-45 (Spirit/Insight/Lore era, up to Sol/Shael/Dol)
| runeword | runes | base | V effects | use |
|---|---|---|---|---|
| **Spirit** | Tal Thul Ort Amn | `swor,shld` 4 | +2 skills, FCR 25-35, mana 89-112, vit +22, FHR 55 (`balance3`), abs magic 3-8, ac-miss | **Best mid-game caster item for Sorc/Necro/Druid/Assassin(?)** and Paladins; sword in main hand (Broad Sword) or shield (Monarch, or Paladin shield for Paladin). Both parts want one set of runes each, so 8 runes for both. Amn clvl 25 |
| **Insight** | Ral Tir Tal Sol | `pole,staf` 4 | Meditation aura 12-17, IAS 35 (`cast2`), +200-260% ED, all stats +5, crit strike | **Merc weapon** (Desert Merc uses spears/polearms, V `docs/progression/merc.md`); the aura applies to the player. Sol clvl 27 |
| **Lore** | Ort Sol | `helm` 2 | all skills +1, light +2, mana per kill 2, +10 energy | caster helm until Harlequin/Andariel; cheap |
| **Smoke** | Nef Lum | `tors` 2 | res all 50, +75% def, FHR 20, -1 light | NM armor, needs Lum (clvl 37) |
| **Lionheart** | Hel Lum Fal | `tors` 3 | +15 str, +20 vit, +15 dex, +50 life, **res all 30**, +20% ED | tanky; Hel clvl 0, Lum 37, Fal 41 |
| **King's Grace** | Amn Ral Thul | `swor,scep` 3 | melee | Paladin weapon option, not needed |
| **Honor** | Amn El Ith Tir Sol | `mele` 5 | +160% ED, +10 str, regen 10, +1 skills | Barb/Paladin 5-os base (Larzuk) |
| **Wealth** | Lem Ko Tir | `tors` 3 | gold +250%, MF 100 | the bot's economy item? gold find only matters if the bot is gold-limited: Lem is clvl 43; marginal |
| **Radiance** | Nef Sol Ith | `helm` 3 | mana 33, vit 10, def +75% | cheap alt helm |

### Stage C, Nightmare/Hell clvl 45-70 (needs Lum..Ber)
- **Fortitude** (El Sol Dol Lo, `weap,tors`): armor +200% def, +300% weapon ED, FCR 25, all res 25-30, life/level; Lo clvl 59. Armor for everyone including the merc.
- **Heart of the Oak** (Ko Vex Pul Thul, `staf,mace`, 4 sockets): +3 all skills, FCR 40, mana 15%, res all 30-40, Oak Sage + Ravens charges; Vex clvl 55. Sorc/Druid/Necro staff.
- **Call to Arms** (Amn Ral Mal Ist Ohm, `weap` 5): V props not copied; Battle Orders; a swap weapon for any BO party; marginal for the bot unless it has an alt-slot swap.
- **Bone** (Sol Um Um, `tors` 3): +2 necro skills, Bone Spear/Bone Armor on hit, mana 100-150; **Necromancer**.
- **Treachery** (Shael Thul Lem, `tors` 3): Venom on hit, Fade 5% on get-hit, **IAS 45**, +2 assassin skills (`ass`): **Assassin**. **Duress** (Shael Um Thul, tors 3): +150-200% def, open wounds, cold dmg.
- **Enigma** (Jah Ith Ber, `tors` 3): Teleport (oskill 1), +2 all skills, +45% FRW, str/level, **end goal**; Jah clvl 65, Ber 63.
- **Infinity** (Ber Mal Ber Ist, `pole`): Conviction aura 12, ED 255-325, pierce lightning: **Merc weapon**, flagged ladder (V).
- **Chaos** (Fal Ohm Um, `h2h`): Assassin claws. **Fury** (Jah Gul Eth, `mele`): Barb. **Harmony** (Tir Ith Sol Ko, `miss`): Amazon bow, Vigor aura (S/U), clvl 39. **Melody** (Shael Ko Nef, miss), **Wrath/Faith/Ice** (Amazon later).
- **Dragon** (Sur Lo Sol, `tors,shld`), **Dream**, **Phoenix**, **Exile's Path** (Vex Ohm Ist Dol, paladin shield) - all late.

### Per-class pick (stage A -> C), all U for the ordering, V for rune lists
- **Sorceress**: Stealth armor, Leaf staff (fire) at stage A; Spirit sword + Spirit shield at B (the bot has no swap, so the weapon slot wants the staff: prefer Spirit **shield**-slot? a Monarch needs Str 156, so the Sorc shield slot stays AP/Rhyme; the Spirit sword replaces the staff, the leveler must decide) - U; AP shield 3os; Lore helm; stage C Heart of the Oak, Enigma.
- **Necromancer**: Stealth, Spirit sword (wand slot -> weapon), Lore, AP; C: Bone armor, Heart of the Oak, Enigma.
- **Paladin**: Stealth, Steel (sword), AP shield with Paladin shield 4-os... AP needs exactly 3 sockets; Paladin bases are max-4 (Heraldic) so use Rhyme (2) or Spirit (4). B: Spirit sword + Spirit paladin shield (4 sockets), Insight not needed; C: Fortitude, Enigma, Exile's Path.
- **Amazon**: Stealth, Zephyr bow (A), Harmony bow (B), Edge/Faith later; AP for shields (javelin) - bow amazons have a quiver in the left hand (V `docs/gear.md`): no shield.
- **Druid**: Stealth, Spirit sword, Lore, AP; C: Heart of the Oak staff, Enigma.
- **Assassin**: Stealth, Steel? (claw not allowed), Treachery armor, Chaos claw (late).
- **Barbarian**: Steel, Strength, Honor (5os), Stealth, AP; C: Fortitude, Fury.
- **All with a mercenary**: Insight polearm for the Act 2 merc (B), Fortitude/Treachery armor for merc, Lionheart/Duress helm... Merc helm is a plain unique/rare circlet; runeword helms for mercs: Lore/Nadir are fine. Merc gear policy in `docs/progression/merc.md`.

## 6. Gold budget (V tables for prices, U for formulas not found)

- **Belt**: no belt = 4 slots (V `belt.Slots`), Sash/Light Belt 8, Belt/Heavy Belt 12, Plated Belt and all exceptional/elite belts 16 (V belts dump). The default plan splits columns hp,hp,mp,hp (rows of the belt count per column, 8 slots = 2 per column). Tier cap: clvl 1-12 light potions, 13-30 up to Strong (tier 4), 31+ Greater (tier 5), 31+ one Rejuvenation column (V plan.json). Buy a **Belt at clvl ~12** (495 base) to double capacity: cheapest meaningful upgrade.
- **Potion prices (V base costs, vendor price differs, U)**: HP 30/75/125/250/500; MP 60/150/300/500/1000; rejuv 400, full rejuv 1500. A full belt of 4 columns x 3 rows (12 slots) of grade 4: 3 x 250 x ... ~ 3000 gold per refill: this is why a bot with 3000 gold at clvl 25 can't refill everything every trip. The `LowGoldBelow 2000` knob (V) already shifts to selling mode.
- **Merc revive (V Maxroll, https://maxroll.gg/d2/resources/mercenary)**: `min(int(hlvl*hlvl/2)*15, 50000)`. Examples: hlvl 20 = 3000, 30 = 6750, 40 = 12000, 50 = 18750, 57 = 24360 (V from the page), cap 50000. Merc level normally = clvl-ish. Policy: **reserve the revive cost before spending on anything non-essential**; if gold < revive cost and the merc is dead, the planner inserts a gold farm (already in merc.md). A merc is worth it only while hlvl <= ~40 (12000); beyond that skip reviving in Normal (consider gear) and use the other-act merc at full price once hired. Hire cost: V live Palli clvl 13 row cost 360 (one click).
- **Repair**: Blacksmith (Charsi etc.) "Repair All"; formula not found (U): pay when below `RepairBelowPct`. Rune repair only for emergencies (4.1). Avoid ethereal items (cannot be repaired by cube).
- **First purchases order** (U, derived): TP tome (450 V live) > ID tome > Belt/Light Belt > potions to the tier cap > repair > weapon/armor shop upgrades > skip gambling. Do not buy gems (base values 500-30000).
- **Stash/sell**: sell magic/rare/white junk to the NPC of the same town trip (`ChorePlan` order heal, identify, sell, potions, repair, stash, gold - V `chores.go`). Gold stash limit in classic: 100000 in the stash and carry limit clvl*10000 (the pickit `gold` needs rule uses `Level*10000`, V) - U.

## 7. Identification (V code, S/U game)
- **Cain**: free identify of all items in the pack (talk to him; he was rescued in Act 1 after Tristram quest), plus Act 1-5 availability: only Act 1 (Rogue Encampment) after rescue in 1.14d (U; the D2 Cain also stands in Harrogath? no). The code does not use him (dialog unmapped, V `identify.go`). Package P5: map Cain's dialog (live).
- **Scroll/Tome** (V code flow, UNVERIFIED live): tome `ibk` base 200, scroll `isc` base 80. A tome holds 20 scrolls (S/U). Use scrolls only on items the pickit marked `needId` (jewelry now, soon charms/uniques/rares of interest).
- Items in the cube are identified by Cain for free in the **original** game text (S Arreat cube page: "Items will be Identified by Cain while inside the Horadric Cube"); I believe this is not true in 1.14d (U). Skip.
- Policy: identify **rings, amulets, circlets, charms with ilvl >= 20 (clvl >= 20), unique/set**; do **not** identify rare weapons/armor except as upgrade candidates (score unknown bonus 20 is low).

## 8. How to rate gear from stat reads (design for `internal/leveler/score.go`, V = existing code)
Existing V behaviour is in section 0. Changes ranked by value:
1. **Class weights** (U values, tuning): Sorc/Necro/Druid: FCR to breakpoint (Sorc FCR breakpoints 9, 20, 37, 63, 105, 200 per U memory; Necro/Druid breakpoints differ), life, resist all, +skills, mana. Melee: IAS, ED, life/vit, res, speed. Archer: IAS, ED, +bow skills.
2. **Resist cap awareness**: the sum of each resist is capped 75 (cap -60 Normal, -40 NM, 0 Hell penalty: Normal 0, NM -40, Hell -100: S common knowledge, U verified). Adding resists is worth full weight until the cap including difficulty penalty, 0 after. The scorer gives 1.2 flat now: it overvalues res stacking.
3. **Runeword items**: runeword item stats come from the item; if stats unreadable and `FlagRuneword` is set, use a table of the named runeword's V properties from `data.Runewords` (name unavailable in the memory read: U).
4. **Ethereal**: replace the +25% by a durability penalty for non-repairable items.
5. **Required level and strength**: `Cand.Usable/Why` already handles (V); no change.
6. **Quality ordering** `unknownBonus`: uniques get 30 only if the base is not junk; compare to base unique list when `FileIndex` is readable.
7. **Sets**: treat a set item as unique+5 until set bonus can be counted (needs equip list: U).

## 9. Open questions
- Q1: Do ladder-flagged runewords and Hel+ rune upgrades work in offline 1.14d SP? (U, I believe yes; test with a Hel-rune upgrade or Spirit with natural drops.) Changing `RunewordAvailableOffline` depends on this.
- Q2: Larzuk socket count in 1.14d (Normal): max-for-base vs ilvl cap (S D2R page); verify on first Normal Larzuk.
- Q3: Vendor sell and repair formulas: calibrate with a live log (P9).
- Q4: Cain dialog and whether the tome of identify price is 200-base (V live number needed).
- Q5: `FileIndex` meaning of unique/set (needed for named unique rules).
- Q6: Whether Repair All repairs ethereal gear and whether ethereal is a net positive for the bot.
- Q7: Tomes: TP tome 450 (V live) vs base 250; potions' vendor price vs base cost.
- Q8: Rune drop rates per act/area (Countess etc.) - no V source fetched; plan Countess runs by `docs/goals/planner.md` anyway.

## 10. Prioritised work packages for d2-worker agents (small, offline unless noted)

| # | package | owns | acceptance |
|---|---|---|---|
| P1 | Pickit data update: exact 2/3/4-socket white bases, white base list from 1.5, charm stat filters, gem keep counts hook | `internal/loot/pickit_starter.json`, `loot_test.go` | table-driven tests per item class; `go test ./internal/loot` |
| P2 | Cube planner (pure): from stash+pack + `data.Cube`, decide which recipes to run (rune ladder to a target rune, gem ladder, rejuv from spare potions), output `[]Recipe` for `town.Transmute`; honour `Ladder` flag setting | new `internal/cubeplan` | `mem.Fake`/table tests: e.g. 7 Tal -> 2 Ral; never consumes runes a target runeword needs |
| P3 | Runeword planner (pure): given owned runes, white bases with socket counts, class, clvl, return makeable runewords by priority from section 5 (data-driven JSON `runewords_plan.json` with class/stage) | new `internal/runeword` | tests for Stealth/Spirit/Insight/AP/Lore selection, wrong socket count rejected |
| P4 | Larzuk reward chore + Act 5 quest hook (socket the best base) | `internal/quest`/`town` | needs live: d2-live package |
| P5 | Rune insertion UI (cursor rune onto socketed item), verify FlagRuneword | `internal/town` | d2-live: Stealth with Tal+Eth |
| P6 | Verify `FileIndex` and unique table lookup; named unique/set rules | `internal/d2`, `internal/data` | live read of a dropped unique |
| P7 | Merc revive: coded chore + budget reserve `min(int(lvl*lvl/2)*15, 50000)` | `internal/town/merc.go`, tuning | pure tests of the formula/reserve |
| P8 | Belt purchase policy (buy Belt at clvl 12), potion tier by gold, spending order table | `internal/town`, `belt/plan.json` | pure tests |
| P9 | Calibration logger: record vendor sell price and repair price per item to a file; fit `SellValue` | `internal/loot`, `cmd/townlive` | live run, report table |
| P10 | Live probe for Q1: try a ladder-flagged recipe offline (e.g. rune upgrade Dol->Hel) when runes are available; update `RunewordAvailableOffline` | `internal/data` | d2-live |
| P11 | Gear scorer: class weights, resist cap, runeword awareness, ethereal penalty | `internal/leveler/score.go` | table tests against section 8 |
| P12 | Cain dialog mapping | `internal/town` | d2-live |

Order: P1 -> P3 -> P2 -> P7 -> P8 -> P11 offline first; P10/P5/P9 live next; P4/P12/P6 later.

## 10. Cube chore (package cubejobs, town/cubejobs.go) - offline only

- `internal/cubejobs.Plan(counts, Opts)` (pure) returns up to MaxJobs transmutes: rune upgrades lowest first (3 -> 1 to Pul, 2 -> 1 from Um, gem per 4.2, gem table V from this section's recipe list), then gem grade upgrades (3 same -> next). Keep protects codes; RuneCap (default 10 Thul) and GemCap (default standard) bound outputs. Not done: rejuvenation recipes (need potion-grade matching), TP/ID tomes (no cube recipe).
- `Town.Chores` step "cube" (`ChorePlan.Cube`, set by the d2bot adapter only when `Tuning.Bot.CubeJobs` is on and the visit stashes; knob default OFF, CubeMaxJobs 6). Flow: open stash, count pack+stash, plan, per job pull inputs from the stash to the pack, close panels, `Transmute`, check the result code, stop at the first failure; finally park gems/runes back in the stash. Gold is never touched.
- Verified: simulator tests only (`go test ./internal/town -run CubeJobs` on the worker host, sim recipes mirror the planner so they prove the flow, not the game). V live: `Transmute`. U: stash items readable with the stash panel closed (code reads them with it open), pack space for 3-4 inputs, upgrade recipes in 1.14d offline (Q1 for Dol->Hel and up), cube drop of Transmute result into the pack with gem recipes, that the pickit keeps chipped/flawed gems.
- Live probe checklist (one character, Cube in pack, stash with 3 chipped amethyst `gcv` and 3 `r01`): set `bot.CubeJobs` true, run `townlive` chores with `ChorePlan{Cube:true}`; expect `gfv` and `r02` in the stash, pack unchanged, no panel open. Then with 3 Thul + 1 chipped topaz and RuneCap 11 expect Amn.

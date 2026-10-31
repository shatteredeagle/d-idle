# SME class file: Paladin (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companions: `builds-leveling.md`, `builds-endgame-melee.md`, `builds-gear.md`, `builds-open-questions.md`, `docs/classes/paladin.md`. Feeds an unattended bot with no respec (`docs/leveler.md`).
Tags: **V** fetched or computed from the install's tables (URL or "T"), **V1** carried from first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary, **U** memory or judgement. D2R-only facts are marked "D2R".

## 0. Verdict

The Paladin is the best all-round class for an unattended bot: highest base life of the hybrids (+2 per clvl, +3 per Vit, T), 30 block factor, auras that help the mercenary and himself, and very few immunities for the Hammerdin line because Blessed Hammer deals **magic** damage and only 6 distinct spawn types in all of Hell are magic-immune (T). The costs: the first 17 clvls have no good ranged attack, so the no-respec path here uses Zeal (melee, 4 points) as the early engine and then pours everything into Blessed Hammer; its 17 points arrive only at clvl 40. Holy Bolt is NOT a general attack in 1.14d (it damages only hostile undead, V AB `title=Holy_Bolt`), which is why the common "Holy Bolt first" idea does not apply.

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 25, Dex 20, Int 15, Vit 25 |
| Life | +2 per clvl, +3 per Vit; Mana +1.5 per clvl, +1.5 per Energy; Stamina +1/+1 |
| Starting life/mana (U) | about 55 / 15 |
| Starting equipment | Short Sword, Buckler, 4 minor healing potions, TP and ID scrolls (T codes ssd, buc, hp1 x4, tsc, isc) |
| Block / AR | block factor 30 (best), ToHitFactor 20 per Dex |

Points: 1 per clvl from 2 plus 4 per difficulty; 47 at clvl 40 on our timeline. Stat points 5 per clvl.

## 2. Skills, synergies, per-level numbers (T)

Added damage per hit unless marked /s. Mana shown at skill level 1/10/20. Auras have no mana cost.

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Sacrifice | 1 | - | - | - | - |
| Smite | 1 | - | 2/2/2 | - | - |
| Might | 1 | - | 1/1/1 | - | - |
| Prayer | 1 | - | 1/2.7/4.6 | - | - |
| Resist Fire | 1 | - | - | - | - |
| Holy Bolt | 6 | - | 2/2.6/3.2 | 8-16 / 84-94 / 196-220 mag | +50%/lvl: Blessed Hammer, Fist of the Heavens |
| Holy Fire | 6 | Might | - | 1-3 / 6-8 / 18-20 fire | +18%/lvl: Resist Fire, Salvation |
| Thorns | 6 | - | - | - | - |
| Defiance | 6 | - | - | - | - |
| Resist Cold | 6 | - | - | - | - |
| Zeal | 12 | Sacrifice | 2/2/2 | - | - |
| Charge | 12 | Smite | 9/9/9 | - | - |
| Blessed Aim | 12 | Might | - | - | - |
| Cleansing | 12 | Prayer | - | - | - |
| Resist Lightning | 12 | - | - | - | - |
| Vengeance | 18 | Zeal | 4/6.2/8.8 | - | - |
| Blessed Hammer | 18 | Holy Bolt | 5/7.2/9.8 | 12-16 / 88-92 / 196-200 mag | +14%/lvl: Vigor, Blessed Aim |
| Concentration | 18 | Blessed Aim | - | - | - |
| Holy Freeze | 18 | Holy Fire | - | 2-3 / 13-14 / 37-38 cold | +15%/lvl: Resist Cold, Salvation |
| Vigor | 18 | Cleansing, Defiance | - | - | - |
| Conversion | 24 | Vengeance | 4/4/4 | - | - |
| Holy Shield | 24 | Charge, Blessed Hammer | 35/35/35 | - | - |
| Holy Shock | 24 | Holy Freeze | - | 1-10 / 1-68 / 1-156 ltng | +12%/lvl: Resist Lightning, Salvation |
| Sanctuary | 24 | Thorns, Holy Freeze | 1/1/1 | 8-16 / 44-54 / 88-108 mag | +7%/lvl: Cleansing |
| Meditation | 24 | Cleansing | - | - | - |
| Fist of the Heavens | 30 | Blessed Hammer, Conversion | 25/25/25 | 150-200 / 315-365 / 675-725 ltng | +7%/lvl: Holy Shock |
| Fanaticism | 30 | Concentration | - | - | - |
| Conviction | 30 | Sanctuary | - | - | - |
| Redemption | 30 | Vigor | - | - | - |
| Salvation | 30 | - | - | - | - |

Formulas (V AB `https://theamazonbasin.com/wiki/index.php?title=<Skill>`, consistent with T):
- **Blessed Hammer**: magic damage min 8*slvl+4 / 10*slvl-12 / 12*slvl-44 / 13*slvl-66 / 14*slvl-94 for slvl 1-8 / 9-16 / 17-22 / 23-28 / 29+ (max 4 higher at low levels), mana 4.75 + 0.25*slvl, +14% magic damage per Blessed Aim and Vigor level (V AB, T Param8 14). Our T table at level 20: 196-200 base. The hammer circles outward and hits most things in a spiral; bonus vs undead (S).
- **Zeal** (clvl 12, prereq Sacrifice): 100% weapon damage, +10% AR per level, damage bonus max(6*slvl-24, 0)% plus synergy from Sacrifice (V AB; T calc: hits = min(2+lvl-1, 5), so **level 4 is the cap for hits**, extra levels add only damage). Mana 2.
- **Holy Bolt**: damages **only hostile undead** and heals allies (V AB); +50% magic damage to undead per Blessed Hammer level (V AB: "50 * (Blessed Hammer + Fist of the Heavens)"). Mana 1.94 + 0.0625*slvl.
- **Holy Shield** (clvl 24, prereq Charge + Blessed Hammer): mana 35, lasts 25*slvl+5 seconds, +15*slvl+10% defense, plus 15% per Defiance level (V AB).
- **Vigor**: movement speed bonus min(7 + 43*(110*slvl/(slvl+6))/100, 50)%, +25+25*slvl% stamina (V AB); **Blessed Aim** +60+15*slvl% attack rating, passive +5% per level to itself; **Concentration** radius (28+4*slvl)/3 yards (V AB). **Fanaticism** (clvl 30): +33+17*slvl% damage, allied half (V AB).
- **Might**: aura +40% damage at level 1 plus 10% per level (T Param3/4 40/10; V1 for Maxroll use as the standard early aura).
- Merc benefit: aura levels apply to the merc and party members within radius (V AS: Desert mercs share auras with all party members; Paladin auras reach party/minions, `docs/classes/party.md`).

## 3. Mana, potions, early game problems

| clvl | Vit points added | Energy points added | life (no gear) | mana (no gear) |
|---|---|---|---|---|
| 10 | 39 | 0 | 190 | 28 |
| 20 | 80 | 0 | 333 | 43 |
| 30 | 100 | 0 | 413 | 58 |
| 40 | 150 | 0 | 583 | 73 |

- Zeal costs 2 mana per use and Blessed Hammer 5-9 (T), so the Paladin is the cheapest class on mana: a 30-60 mana pool is 5-10 casts of BH and Zeal is almost free. Energy points are never needed (stat plan puts none).
- Early (clvl 1-11): melee only with Might and the start Short Sword; Prayer 3 (heals 6+ HP per second while no mana cost, S), Sacrifice 1 (prereq of Zeal). Early life 55 and +2 per clvl (T): the first Act 1 packs are the risk; use the Rogue merc (V AS: level 3 at 100 gold) and melee only with healing potions.
- If potions run out: step back to town; Prayer heals slowly and the merc heals under Prayer (V AS: "The Paladin Prayer aura or Holy Bolt heal hirelings"); never take Act 1 packs with fewer than 2 potions (project `kit_clvl`, V `docs/bossgate.md`).
- Stamina: melee classes run out walking; buy Stamina potions early (V1) and Vigor 6 helps from clvl 22.

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil at clvl 5, Radament 15, Izual 24, NM Den/Radament/Izual at 30/34/38 (U). Machine-validated against `skills.tsv` (reqlevel and prerequisites OK, T). Final allocation at clvl 40: Blessed Hammer 17, Vigor 6, Blessed Aim 5, Might 4, Zeal 4, Prayer 3, Sacrifice 1, Holy Bolt 1, Defiance 1, Smite 1, Charge 1, Cleansing 1, Concentration 1, Holy Shield 1.

| pt | at | skill point | stat points (5/clvl) | Zeal / Blessed Hammer / Vigor / Blessed Aim |
|---|---|---|---|---|
| 1 | clvl 2 | Might | 3 Str, 2 Vit |  |
| 2 | clvl 3 | Sacrifice | 3 Str, 2 Vit |  |
| 3 | clvl 4 | Might | 5 Vit |  |
| 4 | clvl 5 | Prayer | 5 Vit |  |
| 5 | Den of Evil N (clvl ~5) | Prayer | - |  |
| 6 | clvl 6 | Holy Bolt | 5 Vit |  |
| 7 | clvl 7 | Defiance | 5 Vit |  |
| 8 | clvl 8 | Smite | 5 Vit |  |
| 9 | clvl 9 | Might | 5 Vit |  |
| 10 | clvl 10 | Prayer | 5 Vit |  |
| 11 | clvl 11 | Might | 5 Vit |  |
| 12 | clvl 12 | Zeal | 5 Vit | Zeal 1 mana 2.0 |
| 13 | clvl 13 | Zeal | 5 Vit | Zeal 2 mana 2.0 |
| 14 | clvl 14 | Zeal | 5 Vit | Zeal 3 mana 2.0 |
| 15 | clvl 15 | Zeal | 5 Vit | Zeal 4 mana 2.0 |
| 16 | Radament N (clvl ~15) | Charge | - | Zeal 4 mana 2.0 |
| 17 | clvl 16 | Blessed Aim | 5 Vit | Zeal 4 mana 2.0; Blessed Aim 1 |
| 18 | clvl 17 | Cleansing | 5 Vit | Zeal 4 mana 2.0; Blessed Aim 1 |
| 19 | clvl 18 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 1 mana 5.0 14-18 (+14% syn); Blessed Aim 1 |
| 20 | clvl 19 | Concentration | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 1 mana 5.0 14-18 (+14% syn); Blessed Aim 1 |
| 21 | clvl 20 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 2 mana 5.2 23-27 (+14% syn); Blessed Aim 1 |
| 22 | clvl 21 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 3 mana 5.5 32-36 (+14% syn); Blessed Aim 1 |
| 23 | clvl 22 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 4 mana 5.8 41-46 (+14% syn); Blessed Aim 1 |
| 24 | clvl 23 | Vigor | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 4 mana 5.8 46-51 (+28% syn); Vigor 1; Blessed Aim 1 |
| 25 | clvl 24 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 5 mana 6.0 56-61 (+28% syn); Vigor 1; Blessed Aim 1 |
| 26 | Izual N (clvl ~24) | Vigor | - | Zeal 4 mana 2.0; Blessed Hammer 5 mana 6.0 62-68 (+42% syn); Vigor 2; Blessed Aim 1 |
| 27 | Izual N 2 (clvl ~24) | Blessed Hammer | - | Zeal 4 mana 2.0; Blessed Hammer 6 mana 6.2 74-80 (+42% syn); Vigor 2; Blessed Aim 1 |
| 28 | clvl 25 | Vigor | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 6 mana 6.2 81-87 (+56% syn); Vigor 3; Blessed Aim 1 |
| 29 | clvl 26 | Holy Shield | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 6 mana 6.2 81-87 (+56% syn); Vigor 3; Blessed Aim 1 |
| 30 | clvl 27 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 7 mana 6.5 94-100 (+56% syn); Vigor 3; Blessed Aim 1 |
| 31 | clvl 28 | Vigor | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 7 mana 6.5 102-109 (+70% syn); Vigor 4; Blessed Aim 1 |
| 32 | clvl 29 | Blessed Hammer | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 8 mana 6.8 116-122 (+70% syn); Vigor 4; Blessed Aim 1 |
| 33 | clvl 30 | Vigor | 3 Str, 2 Vit | Zeal 4 mana 2.0; Blessed Hammer 8 mana 6.8 125-132 (+84% syn); Vigor 5; Blessed Aim 1 |
| 34 | Den of Evil NM (clvl ~30) | Blessed Aim | - | Zeal 4 mana 2.0; Blessed Hammer 8 mana 6.8 135-143 (+98% syn); Vigor 5; Blessed Aim 2 |
| 35 | clvl 31 | Blessed Aim | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 8 mana 6.8 144-153 (+112% syn); Vigor 5; Blessed Aim 3 |
| 36 | clvl 32 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 9 mana 7.0 165-174 (+112% syn); Vigor 5; Blessed Aim 3 |
| 37 | clvl 33 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 10 mana 7.2 187-195 (+112% syn); Vigor 5; Blessed Aim 3 |
| 38 | clvl 34 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 11 mana 7.5 208-216 (+112% syn); Vigor 5; Blessed Aim 3 |
| 39 | Radament NM (clvl ~34) | Vigor | - | Zeal 4 mana 2.0; Blessed Hammer 11 mana 7.5 221-231 (+126% syn); Vigor 6; Blessed Aim 3 |
| 40 | clvl 35 | Blessed Aim | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 11 mana 7.5 235-245 (+140% syn); Vigor 6; Blessed Aim 4 |
| 41 | clvl 36 | Blessed Aim | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 11 mana 7.5 249-259 (+154% syn); Vigor 6; Blessed Aim 5 |
| 42 | clvl 37 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 12 mana 7.8 274-284 (+154% syn); Vigor 6; Blessed Aim 5 |
| 43 | clvl 38 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 13 mana 8.0 300-310 (+154% syn); Vigor 6; Blessed Aim 5 |
| 44 | Izual NM (clvl ~38) | Blessed Hammer | - | Zeal 4 mana 2.0; Blessed Hammer 14 mana 8.2 325-335 (+154% syn); Vigor 6; Blessed Aim 5 |
| 45 | Izual NM 2 (clvl ~38) | Blessed Hammer | - | Zeal 4 mana 2.0; Blessed Hammer 15 mana 8.5 351-361 (+154% syn); Vigor 6; Blessed Aim 5 |
| 46 | clvl 39 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 16 mana 8.8 376-386 (+154% syn); Vigor 6; Blessed Aim 5 |
| 47 | clvl 40 | Blessed Hammer | 5 Vit | Zeal 4 mana 2.0; Blessed Hammer 17 mana 9.0 406-417 (+154% syn); Vigor 6; Blessed Aim 5 |

Stat plan: Vit from clvl 4; 3 Str / 2 Vit per level from clvl 18 to 30 to wear heavier armor and shields (Str 25 start, +about 40 = 65 enough for most Act 5 body armors, U), Vit again from 31. Dex 0 (the 30 block factor is enough, V `docs/classes/paladin.md` did not require Dex).
Why: Might 4 + Prayer 3 give offense and sustain at clvl 2-11; Sacrifice 1 and Zeal 4 (the hits cap) are the early engine (clvl 12-17); Smite 1 + Charge 1 are prerequisites of Holy Shield; Blessed Aim and Vigor feed BH by +14% each per level (T Param8) so they carry 11 points; Concentration 1 is the aura that protects BH from interruption (S) and boosts damage (V1 Maxroll Hammerdin); BH 17 by clvl 40 (last table row has the exact number). Humans do Holy Fire -> respec at 19 into BH (V1 Maxroll) and put BH 20 right after clvl 19; the bot loses about 10-20% of the early power by not doing so (U).

## 5. Mercenary

- Act 2 Desert merc (V AS/T): Combat Prayer / Thorns / Prayer, Defense Defiance / Holy Freeze / Defiance, Offense Blessed Aim / Might / Blessed Aim by difficulty. For a Paladin who already runs Concentration/Vigor, the merc's aura is an extra stackable aura of a different type: Might (NM) adds damage, Defiance/Prayer adds survival. Best bot pick: Defiance or Prayer (Normal and Hell), Might in NM (U).
- Rogue (Act 1) for clvl 1-12; Act 5 Barbarian has Bash/Stun and the most life (V AS).
- Merc weapon: spear/polearm; no Infinity offline (ladder, V `builds-open-questions.md` Q3).

## 6. Gear per act on a budget

| stage | weapon | armor / helm / shield | rest | notes |
|---|---|---|---|---|
| Normal Act 1-2 | start Short Sword; buy a sword/mace with +fire or crushing (S); Malice (Ith+El+Eth, T) or Steel (Tir+El) in a 3-socket weapon (V1: steel scimitar clvl 13) | Stealth (Tal+Eth) in 2os armor at clvl 17 (V1); a paladin Targe/Buckler with sockets | belt 12 slots | Paladin-only shields (Zakarum shields) carry +skills (S) |
| Normal Act 3-5 | King's Grace (Amn+Ral+Thul, sword/scepter: +100% ed, +150 AR, +100% vs demons, +50% vs undead, T) once Ral is found (Act 5 quest gives Ral, V `builds-open-questions.md` Q1) | Lore helm (Ort+Sol, T); Smoke (Nef+Lum body armor, T) | Rhyme (Shael+Eth shield: +20% block, +25 res all, T) | resist gems |
| Nightmare | Honor (Amn+El+Ith+Tir+Sol, any melee: +160% ed, +1 all skills, T) or Fury later | Chains of Honor (+2 all skills, +50 res with Um = 65 total, V open-questions Q2); Lionheart (Hel+Lum+Fal, T) | Fanaticism-ready shield | raw res 60 each by NM entry |
| Hell | Hand of Justice (Holy Fire, T) for melee; BH uses any weapon with +skills | Enigma (+2 skills, Teleport), Chains of Honor | Harlequin Crest (S) | raw res 125 each |

Ladder-only skip list for the Paladin (T `server`=1): Spirit, Insight, Fortitude, Dream, Dragon, Phoenix, Faith, Ice, Lawbringer, Oath, Obedience, Pride, Wrath, Grief, Voice of Reason, Last Wish, Death, Destruction, Rift, Harmony, Brand, Edge, Infinity.

## 7. Resists and immunities

Targets: Normal 0 needed; Nightmare entry raw 60 fire/cold/lightning (+20 after -40); Hell raw 125 minimum (+25 after -100), cap 175 (V AS). Paladin gets resist auras (Resist Fire/Cold/Lightning, Salvation at clvl 30, V AS resistances page), but they cost a point each per level and a bot can only keep one aura active at a time except via the merc; gear is the main source (U).

Immunity landscape (T; NM / Hell types):
| act | distinct spawn types | physical imm NM / Hell | magic imm NM / Hell | fire imm NM / Hell | lightning imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 0 | 0 / 16 | 0 / 14 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 0 / 3 | 1 / 13 | 0 / 11 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 2 | 0 / 10 | 0 / 15 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 0 / 0 | 1 / 4 | 1 / 8 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 0 / 1 | 2 / 11 | 4 / 8 |

- Hell magic-immune spawn types by act:
  - Act 2: Plague Bearer, Guardian, Unraveler
  - Act 3: Horadrim Ancient, Wailing Beast
  - Act 5: Pit Lord

How humans handle it: BH is magic damage, so the Hammerdin hits almost everything; the only magic-immune regular spawn types in Hell are Plague Bearer, Guardian, Unraveler (Act 2), Horadrim Ancient, Wailing Beast (Act 3), Pit Lord (Act 5) (T). Players skip those packs, use Holy Fire/Fanaticism melee, or Fist of the Heavens (lightning) on a respec (V1 Maxroll). Magic immune bosses: none among Act bosses (U). Physical immunes (T: 2-4 types per Hell act) only hurt the melee Zealot line.

## 8. Strengths, weaknesses, deaths by act/difficulty

- Normal Act 1: strong. Melee with Might and Prayer, Charge later; weakest at clvl 1-6 (55 life). Deaths: Blood Raven's arrows and Fire Blast, Andariel's poison cloud if the merc is dead (U).
- Normal Act 2-3: Holy Bolt dead end (undead-only, V AB); BH not yet available before 18. Duriel (cold) and Mephisto lightning are the first real walls; Holy Shield at 24 (35 mana) makes the Paladin much safer (V AB defense bonus).
- Normal Act 4-5: Diablo's fire nova, Baal waves; BH at 12+ clears. Strong.
- Nightmare: resist -40 with 400-500 life (T table); Fire/Lightning enchanted mobs, Mephisto and Izual one-shot unprepared chars (U).
- Hell: magic-immune packs are rare, physical/elemental dangers dominate; Hell resists -100 (V AS) so raw 125+ is required.
- Typical bot deaths (U, predicted): hammers cast when a pack surrounds the Paladin without Holy Shield; Zeal into fire-enchanted/cursed packs; dying with Act 5 Barbarian soldiers when the merc is dead.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | no-respec fit |
|---|---|---|---|---|
| Hammerdin (this file) | Blessed Hammer, Concentration, Vigor, Blessed Aim, Holy Shield | highest clear speed, magic damage (almost no immunes), cheap mana | needs BH-aimed positioning, weak at clvl 1-17 without respec | best late, slow early |
| Zealot / Fanatic (Zeal -> Fanaticism) | Sacrifice, Zeal 20, Vengeance, Fanaticism at 30 | no respec needed, strong early melee, big damage with gear (V1) | physical immunes (T: 0 in NM, 2-4 types per Hell act); needs IAS gear; melee deaths | simplest bot, safest to level |
| Smiter / Fist of the Heavens | Smite, Holy Shield, Fist of the Heavens at 30, Holy Bolt synergy | high boss damage | block/Dex gear, lightning immune types (T: 8-15 per Hell act) | not as a sole line |

## 10. When is the character "ready" (bot checks; U unless tagged)

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 6-8, Might 3+, 4 health potions, TP tome | merc hired (Rogue) |
| Andariel | clvl 15-17, life >= 280 (T: about 275 at clvl 15 without gear), Zeal 4 | 6 health potions, Prayer 3 |
| Duriel / Act 3 | clvl 21-23, life >= 380, BH 2+ and Holy Shield soon, cold res >= 20 | Concentration |
| Mephisto | clvl 27-29, life >= 430, BH 6+, all res >= 40 | Holy Shield 1 (clvl 24) |
| Diablo / Baal | clvl 30-34, life >= 480-550, BH 10+ | all res >= 50-60 |
| Enter Nightmare | clvl >= 38-40, raw res >= 60, life >= 560 (T path clvl 40), BH >= 15, Holy Shield 1 | merc alive, 6 health + 2 mana potions |
| Enter Hell | clvl >= 64 (project gate V `docs/progression/difficulty.md`), raw res >= 125, life >= 1100, BH 20+ | potions + rejuvs, magic-immune-pack skip list |

## 11. Open questions

1. Holy Bolt's undead-only damage is V from the Amazon Basin summary (page text truncated at "In LoD bolts only damage hostil..."); confirm in game or via `skills.tsv` flags (U).
2. Real Hammerdin FCR breakpoints for the Paladin (U): the first-pass tables list Sorc and Necro/Druid only.
3. Starting life/mana for all classes are U.

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).


## BUILDV2 implementation note (2026-10-30, fresh Paladin)
Table `paladin-normal30-v2`: Might 2, Sacrifice 1 (clvl 3, only the Zeal prerequisite, V skills.txt), Prayer 1 (clvl 4), Might to clvl 11, Smite at 10, Zeal 4 points at clvl 12-15 (Den point absorbed by a second Might at clvl 5). The combat profile no longer lists Sacrifice anywhere (order Zeal, Smite, Attack; plain Attack below clvl 10). A Paladin that owns Holy Fire or Resist Fire keeps the old table. Prayer is bought but not yet used by the aura ladder (U: no key, no logic).

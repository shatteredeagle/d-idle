# SME class file: Amazon (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companion files: `builds-leveling.md`, `builds-gear.md`, `builds-endgame-amazon-farming.md`, `builds-open-questions.md`, project plan `docs/classes/amazon.md`. Feeds an unattended bot: it has no respec yet (`docs/leveler.md`), so section 4 is a no-respec line.
Tags: **V** fetched or computed from the install's tables this session (URL or "T"), **V1** carried from the first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary/snippet, **U** memory or my judgement. D2R-only facts are marked "D2R".

## 0. Verdict in one paragraph

The Amazon is the class whose best items are the ladder-only ones (Faith, Ice, Harmony, Infinity merc weapon, Grief), so in vanilla offline 1.14d she is a gear-poor class: Windforce, Eaglehorn/Titan's Revenge are uniques, not runewords, and Insight/Spirit are out. For a bot the safest line is a **fire bow Amazon (Fire Arrow -> Exploding Arrow)** to clvl ~40, because it is one targeted ranged skill with real damage from level 1, the fire-immune population is tiny before Hell (T: 1 fire-immune type in all of Nightmare Acts 1-4), and she wears good armor. Her mana is the real cost (Exploding Arrow: 5 mana at level 1, 14.5 at level 20, T). The late-game plan (Lightning Fury javelin or Multiple Shot/Strafe bow) needs a respec or a different character, see section 9.

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 20, Dex 25, Int 15, Vit 20 |
| Life | +2 per clvl, +3 per Vit point; Mana +1.5 per clvl, +1.5 per Energy point; Stamina +1/clvl, +1/Vit |
| Starting life (U, not in the extract) | about 50 |
| Starting equipment | Javelin, Buckler, 4 minor healing potions, TP and ID scrolls (T item codes jav, buc, hp1 x4, tsc, isc) |
| Block factor | 25 (shield block needs Dex, S) |
| Attack rating | ToHitFactor 5 per Dex (T) |

Points: 1 per clvl from 2 (clvl 40 = 39 points) plus 4 quest points per difficulty (Den of Evil +1, Radament +1, Izual +2, V1), 8 of them before clvl 40 if Nightmare's Den/Radament/Izual are done, so 47 points at clvl 40 on the timeline of section 4. Stat points: 5 per clvl (V1).

## 2. Skills, synergies, per-level numbers (all T)

Damage columns are the *added* elemental damage of the skill (before synergy and before the weapon damage that Fire/Cold/Exploding Arrow also use at 100%, V AB for Exploding Arrow "Weapon damage 100%"); the synergy column is "+% to that column per skill level of the named skills". Mana is shown at skill level 1/10/20.

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Magic Arrow | 1 | - | 1.5/0.4/0 | - | - |
| Fire Arrow | 1 | - | 3/4.1/5.4 | 1-4 / 21-24 / 63-70 fire | +12%/lvl: Exploding Arrow |
| Inner Sight | 1 | - | 5/5/5 | - | - |
| Critical Strike | 1 | - | - | - | - |
| Jab | 1 | - | 2/4.2/6.8 | - | - |
| Cold Arrow | 6 | - | 3.5/4.6/5.9 | 3-4 / 22-23 / 53-56 cold | +12%/lvl: Ice Arrow |
| Multiple Shot | 6 | Magic Arrow | 4/13/23 | - | - |
| Dodge | 6 | - | - | - | - |
| Power Strike | 6 | Jab | 2/4.2/6.8 | 1-16 / 1-214 / 1-646 ltng | +10%/lvl: Lightning Strike, Lightning Bolt, Charged Strike, Lightning Fury |
| Poison Javelin | 6 | - | 4/6.2/8.8 | 25-38 / 162-181 / 462-512 poison total (S) pois | +12%/lvl: Plague Javelin |
| Exploding Arrow | 12 | Fire Arrow, Multiple Shot | 5/9.5/14.5 | 2-6 / 51-57 / 129-149 fire | +12%/lvl: Fire Arrow |
| Slow Missiles | 12 | Inner Sight | 5/5/5 | - | - |
| Avoid | 12 | Dodge | - | - | - |
| Impale | 12 | Jab | 3/3/3 | - | - |
| Lightning Bolt | 12 | Poison Javelin | 6/8.2/10.8 | 1-40 / 1-160 / 1-380 ltng | +3%/lvl: Lightning Strike, Power Strike, Charged Strike, Lightning Fury |
| Ice Arrow | 18 | Cold Arrow | 4/6.2/8.8 | 6-10 / 72-78 / 216-232 cold | +8%/lvl: Cold Arrow |
| Guided Arrow | 18 | Cold Arrow, Multiple Shot | 8/5.8/3.2 | - | - |
| Penetrate | 18 | Critical Strike | - | - | - |
| Charged Strike | 18 | Power Strike, Lightning Bolt | 4/6.2/8.8 | 1-30 / 1-146 / 1-322 ltng | +10%/lvl: Lightning Strike, Lightning Bolt, Power Strike, Lightning Fury |
| Plague Javelin | 18 | Lightning Bolt | 7/11.5/16.5 | poison, per-frame units (not resolved) | +10%/lvl: Poison Javelin |
| Strafe | 24 | Guided Arrow | 11/11/11 | - | - |
| Immolation Arrow | 24 | Exploding Arrow | 6/10.5/15.5 | 12-23 / 142-153 / 416-427 fire | +10%/lvl: Exploding Arrow |
| Decoy | 24 | Slow Missiles | 19/12.2/4.8 | - | - |
| Evade | 24 | Avoid | - | - | - |
| Fend | 24 | Impale | 5/5/5 | - | - |
| Freezing Arrow | 30 | Ice Arrow | 9/13.5/18.5 | 40-50 / 140-150 / 310-320 cold | +12%/lvl: Cold Arrow |
| Valkyrie | 30 | Decoy, Evade | 25/34/44 | - | - |
| Pierce | 30 | Penetrate | - | - | - |
| Lightning Strike | 30 | Charged Strike | 9/9/9 | 1-25 / 1-125 / 1-295 ltng | +8%/lvl: Charged Strike, Lightning Bolt, Power Strike, Lightning Fury |
| Lightning Fury | 30 | Plague Javelin | 10/14.5/19.5 | 1-40 / 1-240 / 1-580 ltng | +1%/lvl: Charged Strike, Lightning Bolt, Power Strike, Lightning Strike |

Key formulas read from the table (T unless noted):
- **Exploding Arrow** (clvl 12, needs Fire Arrow + Multiple Shot): +12% fire per Fire Arrow level; **Fire Arrow** gets +12% per Exploding Arrow level, so the two feed each other. At level 20 with 20 Fire Arrow: 129-149 base x (1 + 2.4) = about 440-507 added fire plus 100% weapon damage per arrow explosion.
- **Immolation Arrow** (clvl 24, needs Exploding Arrow): 12-23 at level 1, 416-427 at level 20 (before its own synergy), mana 3 only at low level; it leaves a fire patch, strong vs bosses (S).
- **Multiple Shot** mana 4/13/23 at levels 1/10/20: it fires 2-6 arrows (U) and is mana hungry; **Magic Arrow** has no mana cost at level 10+ (0.4 mana), its magic damage does not care about fire immunity (S).
- **Passives**: Critical Strike (clvl 1) and Penetrate (clvl 18, +35% AR then +10/lvl, T Param1/2), Pierce (clvl 30, Param 10/100), Dodge/Avoid/Evade (Param 65/75/65 start chances, T). None adds damage to a fire bow line, so they are skipped (U).

## 3. Mana, potions, early game problems

- Exploding Arrow costs 5 mana at level 1 and 9.5 at level 10 (T); a 90 mana pool lasts 9 shots at clvl 20. Fire Arrow costs 3-5.4 and is the sustainable skill; the bot should swap to Fire Arrow when mana < 25% and potions < 2 (U). Magic Arrow is nearly free at level 10+, so a few points there guarantee "something to shoot" when mana is empty (that is why section 4 puts 6 points in Magic Arrow).
- Stat plan from Maxroll (V1): Vit to 11, then Dex/Str for gear, Energy 14-16, Vit afterwards. Energy gives only 1.5 mana per point (T), so 3 levels of Energy = about 22 mana; it is optional.
- Run out of potions early (the no-respec scenario): retreat to town (Akara/Lysander/Malah sell minor/light potions, V1), keep the Rogue merc as a damage dealer, never fight the Countess/Andariel below the readiness numbers in section 10. If mana is out and potions are out, melee is useless (Jab only). Use town well/rest between packs.
- Arrows are finite: a stack is 100 to 350 arrows (T `misc.tsv` Arrows minstack 100, maxstack 350; Bolts 60-250), and every bow skill spends ammo (S). The bot needs an ammo rule: carry two stacks, refill from a bow vendor when below ~100 (U which NPCs sell them). Hunter's Bow needs 28 Dex, Short Bow 15, Long Bow 22 Str + 19 Dex, Stag Bow (Amazon only, clvl 14) 30 Dex... (T `weapons.tsv` reqdex column; max sockets 4/3/5/5). The starting javelin needs no ammo for Jab/Power Strike lines (S).

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil taken at clvl 5, Radament at clvl 15, Izual at clvl 24, Nightmare Den/Radament/Izual at clvl 30/34/38 (U, our timeline). The path was machine-validated against `skills.tsv` (every point meets reqlevel and prerequisites, V T). Quest points are the rows labelled with a quest name. Final allocation at clvl 40: Fire Arrow 20, Exploding Arrow 20, Magic Arrow 6, Multiple Shot 1.

Reading guide: "stat points" are the 5 points of that clvl. The right column shows the two focus skills with their mana and added damage including synergy at that moment (T).

| pt | at | skill point | stat points (5/clvl) | Fire Arrow / Exploding Arrow |
|---|---|---|---|---|
| 1 | clvl 2 | Fire Arrow | 5 Vit | Fire Arrow 1 mana 3.0 1-4 |
| 2 | clvl 3 | Fire Arrow | 5 Vit | Fire Arrow 2 mana 3.1 3-6 |
| 3 | clvl 4 | Fire Arrow | 5 Vit | Fire Arrow 3 mana 3.2 5-8 |
| 4 | clvl 5 | Fire Arrow | 5 Vit | Fire Arrow 4 mana 3.4 7-10 |
| 5 | Den of Evil N (clvl ~5) | Magic Arrow | - | Fire Arrow 4 mana 3.4 7-10 |
| 6 | clvl 6 | Multiple Shot | 5 Vit | Fire Arrow 4 mana 3.4 7-10 |
| 7 | clvl 7 | Fire Arrow | 5 Vit | Fire Arrow 5 mana 3.5 9-12 |
| 8 | clvl 8 | Fire Arrow | 5 Vit | Fire Arrow 6 mana 3.6 11-14 |
| 9 | clvl 9 | Fire Arrow | 5 Vit | Fire Arrow 7 mana 3.8 13-16 |
| 10 | clvl 10 | Fire Arrow | 5 Vit | Fire Arrow 8 mana 3.9 15-18 |
| 11 | clvl 11 | Fire Arrow | 5 Vit | Fire Arrow 9 mana 4.0 18-21 |
| 12 | clvl 12 | Exploding Arrow | 3 Dex, 2 Str | Fire Arrow 9 mana 4.0 20-24 (+12% syn); Exploding Arrow 1 mana 5.0 4-12 (+108% syn) |
| 13 | clvl 13 | Exploding Arrow | 5 Str | Fire Arrow 9 mana 4.0 22-26 (+24% syn); Exploding Arrow 2 mana 5.5 15-23 (+108% syn) |
| 14 | clvl 14 | Exploding Arrow | 5 Ene | Fire Arrow 9 mana 4.0 24-29 (+36% syn); Exploding Arrow 3 mana 6.0 25-33 (+108% syn) |
| 15 | clvl 15 | Exploding Arrow | 5 Ene | Fire Arrow 9 mana 4.0 27-31 (+48% syn); Exploding Arrow 4 mana 6.5 35-44 (+108% syn) |
| 16 | Radament N (clvl ~15) | Exploding Arrow | - | Fire Arrow 9 mana 4.0 29-34 (+60% syn); Exploding Arrow 5 mana 7.0 46-54 (+108% syn) |
| 17 | clvl 16 | Exploding Arrow | 5 Ene | Fire Arrow 9 mana 4.0 31-36 (+72% syn); Exploding Arrow 6 mana 7.5 56-64 (+108% syn) |
| 18 | clvl 17 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 33-39 (+84% syn); Exploding Arrow 7 mana 8.0 67-75 (+108% syn) |
| 19 | clvl 18 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 35-41 (+96% syn); Exploding Arrow 8 mana 8.5 77-85 (+108% syn) |
| 20 | clvl 19 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 37-44 (+108% syn); Exploding Arrow 9 mana 9.0 92-102 (+108% syn) |
| 21 | clvl 20 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 40-46 (+120% syn); Exploding Arrow 10 mana 9.5 106-119 (+108% syn) |
| 22 | clvl 21 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 42-49 (+132% syn); Exploding Arrow 11 mana 10.0 121-135 (+108% syn) |
| 23 | clvl 22 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 44-51 (+144% syn); Exploding Arrow 12 mana 10.5 135-152 (+108% syn) |
| 24 | clvl 23 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 46-54 (+156% syn); Exploding Arrow 13 mana 11.0 150-168 (+108% syn) |
| 25 | clvl 24 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 48-56 (+168% syn); Exploding Arrow 14 mana 11.5 164-185 (+108% syn) |
| 26 | Izual N (clvl ~24) | Exploding Arrow | - | Fire Arrow 9 mana 4.0 50-59 (+180% syn); Exploding Arrow 15 mana 12.0 179-202 (+108% syn) |
| 27 | Izual N 2 (clvl ~24) | Exploding Arrow | - | Fire Arrow 9 mana 4.0 53-61 (+192% syn); Exploding Arrow 16 mana 12.5 193-218 (+108% syn) |
| 28 | clvl 25 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 55-64 (+204% syn); Exploding Arrow 17 mana 13.0 212-241 (+108% syn) |
| 29 | clvl 26 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 57-66 (+216% syn); Exploding Arrow 18 mana 13.5 231-264 (+108% syn) |
| 30 | clvl 27 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 59-69 (+228% syn); Exploding Arrow 19 mana 14.0 250-287 (+108% syn) |
| 31 | clvl 28 | Exploding Arrow | 5 Vit | Fire Arrow 9 mana 4.0 61-71 (+240% syn); Exploding Arrow 20 mana 14.5 268-310 (+108% syn) |
| 32 | clvl 29 | Fire Arrow | 5 Vit | Fire Arrow 10 mana 4.1 71-82 (+240% syn); Exploding Arrow 20 mana 14.5 284-328 (+120% syn) |
| 33 | clvl 30 | Fire Arrow | 5 Vit | Fire Arrow 11 mana 4.2 82-92 (+240% syn); Exploding Arrow 20 mana 14.5 299-346 (+132% syn) |
| 34 | Den of Evil NM (clvl ~30) | Fire Arrow | - | Fire Arrow 12 mana 4.4 92-102 (+240% syn); Exploding Arrow 20 mana 14.5 315-364 (+144% syn) |
| 35 | clvl 31 | Fire Arrow | 5 Vit | Fire Arrow 13 mana 4.5 102-112 (+240% syn); Exploding Arrow 20 mana 14.5 330-381 (+156% syn) |
| 36 | clvl 32 | Fire Arrow | 5 Vit | Fire Arrow 14 mana 4.6 112-122 (+240% syn); Exploding Arrow 20 mana 14.5 346-399 (+168% syn) |
| 37 | clvl 33 | Fire Arrow | 5 Vit | Fire Arrow 15 mana 4.8 122-133 (+240% syn); Exploding Arrow 20 mana 14.5 361-417 (+180% syn) |
| 38 | clvl 34 | Fire Arrow | 5 Vit | Fire Arrow 16 mana 4.9 133-143 (+240% syn); Exploding Arrow 20 mana 14.5 377-435 (+192% syn) |
| 39 | Radament NM (clvl ~34) | Fire Arrow | - | Fire Arrow 17 mana 5.0 153-167 (+240% syn); Exploding Arrow 20 mana 14.5 392-453 (+204% syn) |
| 40 | clvl 35 | Fire Arrow | 5 Vit | Fire Arrow 18 mana 5.1 173-190 (+240% syn); Exploding Arrow 20 mana 14.5 408-471 (+216% syn) |
| 41 | clvl 36 | Fire Arrow | 5 Vit | Fire Arrow 19 mana 5.2 194-214 (+240% syn); Exploding Arrow 20 mana 14.5 423-489 (+228% syn) |
| 42 | clvl 37 | Fire Arrow | 5 Vit | Fire Arrow 20 mana 5.4 214-238 (+240% syn); Exploding Arrow 20 mana 14.5 439-507 (+240% syn) |
| 43 | clvl 38 | Magic Arrow | 5 Vit | Fire Arrow 20 mana 5.4 214-238 (+240% syn); Exploding Arrow 20 mana 14.5 439-507 (+240% syn) |
| 44 | Izual NM (clvl ~38) | Magic Arrow | - | Fire Arrow 20 mana 5.4 214-238 (+240% syn); Exploding Arrow 20 mana 14.5 439-507 (+240% syn) |
| 45 | Izual NM 2 (clvl ~38) | Magic Arrow | - | Fire Arrow 20 mana 5.4 214-238 (+240% syn); Exploding Arrow 20 mana 14.5 439-507 (+240% syn) |
| 46 | clvl 39 | Magic Arrow | 5 Vit | Fire Arrow 20 mana 5.4 214-238 (+240% syn); Exploding Arrow 20 mana 14.5 439-507 (+240% syn) |
| 47 | clvl 40 | Magic Arrow | 5 Vit | Fire Arrow 20 mana 5.4 214-238 (+240% syn); Exploding Arrow 20 mana 14.5 439-507 (+240% syn) |

Why this order (U unless tagged): Fire Arrow 4 points give a working ranged attack; Magic Arrow 1 (prerequisite of Multiple Shot) comes with the Den point; Multiple Shot 1 at clvl 6 (prerequisite of Exploding Arrow); Fire Arrow 9 before Exploding Arrow so the first Exploding Arrow point arrives with +108% synergy at clvl 12; Exploding Arrow to 20 by clvl ~32, then Fire Arrow to 20 (synergy), last 5 points Magic Arrow for the fire-immune fallback. Human version (Maxroll, V1): the same Fire Arrow / Exploding Arrow plan with Poison Javelin on swap and a respec at clvl 38 into Lightning Fury/Charged Strike.

## 5. Mercenary

- Act 1 Rogue (V `hireling.tsv`): Fire or Cold Arrow variant; level 3 at 100/150 gold (fire/cold) in Normal, 6000/7500 in Nightmare, 12500/14000 in Hell (T Gold column). Weak: life 45 at level 3 growing +8/lvl (V AS "Level Up: +1 Str, +2 Dex, Defense +6.5, +8 Life") and dies often.
- Act 2 Desert mercenary from Greiz (V): the standard choice. Normal Defiance or Prayer (or Blessed Aim for damage), Nightmare Holy Freeze (slow) or Might, Hell Defiance/Prayer/Blessed Aim. For a bow Amazon, a Defiance/Prayer merc makes the bot survive; Might (NM only) adds melee damage for a spear merc. Merc hire cost 350 gold in Normal (T), 7900 NM, 15000 Hell, level-scaled to your level at hire time (V `hireling.tsv`).
- Act 3 Iron Wolf casters are fragile; Act 5 Barbarian (Bash/Stun) has the most life (T: level 42 NM barb 436 hp vs Act 2 level 43 NM 567 hp).
- Gear for the merc: spears/polearms/javelins and helm/armor (V AS). A Might aura in Nightmare plus Insight is a ladder item; offline use a normal high-damage polearm and Smoke/Stealth-free armor.

## 6. Gear per act on a budget (class-specific; general lists are in `builds-gear.md`)

| stage | weapon | armor/helm/shield | jewelry/belt/boots/gloves | notes |
|---|---|---|---|---|
| Normal Act 1 | start Javelin until clvl 6; buy a Short/Hunter's Bow with 2 Chipped Rubies (V1; Hunter's Bow can take up to 4 sockets, T) | Studded/Quilted leather, Buckler, Cap | none | Charsi can socket via nothing; a 2-socket non-magic Cap/armor drop for Stealth/Lore later (V1) |
| Normal Act 2-3 | Hunter's Bow or Amazon Stag/Ashwood bows from drops (U); socketed magic bow with Rubies | Stealth (Tal+Eth, T runes.tsv) in a 2os armor at clvl 17 (V1), Lore (Ort+Sol) helm | Belt (12 slots) from Fara (V1), Gem-in-helm for fire resist | Staff of Teleportation is a caster item; Amazon uses Teleport charges only by swapping a weapon (bot has no swap) |
| Normal Act 4-5 / Nightmare Act 1-3 | Eaglehorn/Cliffkiller are uniques; Hunter's Bow + 2 Ruby, or Wand of Lower Resist on swap (human only) (V1) | Bone Shield (Rhyme, Shael+Eth) is not Amazon-specific (Rhyme works), Smoke (Nef+Lum) armor, Stealth early | Heavy Gloves of Readiness (IAS, V1), Mammoth belt (V1), Haste boots in NM (V1), resists ring/amulet | NM entry resists: +20 after -40 penalty -> raw 60 each (V `docs/difficulty.md` via V1) |
| Nightmare Act 4-5 / Hell | Windforce (unique bow, U not in tables read), Titan's Revenge javelin (U), Grief and Faith are ladder (skip) | Enigma (Jah+Ith+Ber, any body armor, T), Chains of Honor (Dol+Um+Ber+Ist, +65 all res with Um, T/AB), Lionheart, Harlequin Crest, Arkaine/Skin of the Vipermagi | Raven Frost, Nightwing's/Maraphi's-types, Stone of Jordan | needs Hell drops; not budget |

The weapon constraint is why many humans abandon bows in offline-only play for javelins (Titan's Revenge, Thunderstroke are uniques) or take the Rogue Hell fire plan (S).

## 7. Resists and immunities

Resist targets: Normal 0 needed; Nightmare entry +20 after penalty = raw 60 fire/cold/lightning (the -40 penalty is V AS); Hell +25 after penalty = raw 125 each, cap 75 is raw 175 (V AS; targets V1/project). Poison and physical/magic: not part of the project rule; Dodge/Avoid/Evade do not help against spells (S).

Immunity landscape (T, spawn-list counts; Hell phys/fire/lightning/cold immune types):
| act | distinct spawn types | physical imm NM / Hell | fire imm NM / Hell | lightning imm NM / Hell | cold imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 16 | 0 / 14 | 0 / 11 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 1 / 13 | 0 / 11 | 0 / 13 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 10 | 0 / 15 | 0 / 17 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 1 / 4 | 1 / 8 | 0 / 4 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 2 / 11 | 4 / 8 | 10 / 21 |

- Hell fire-immune spawn types by act:
  - Act 1: Fallen, Fallen Shaman, Carver, Vile Lancer, Carver Shaman, Vile Archer, Returned Mage, Devilkin, Blood Clan, Devilkin Shaman, Bone Mage, Dark One, Dark Shaman, Rat Man, Arach, Ghoul
  - Act 2: Sand Leaper, Saber Cat, Tomb Creeper, Guardian, Hell Slinger, Burning Dead, Sand Raider, Burning Dead Mage, Invader, Blunderbore, Horror Mage, Desert Wing, Hell Clan
  - Act 3: Swamp Dweller, Drowned Carcass, Fetish Shaman, Flayer Shaman, Soul Killer Shaman, Hell Buzzard, Zakarumite, Flame Spider, Spider Magus, Blood Diver
  - Act 4: Doom Knight, Balrog, Pit Lord, Venom Lord
  - Act 5: Demon Rascal, Moon Lord, Demon Trickster, Succubus, Talic, Death Lord, Vile Witch, Greater Hell Spawn, Arach, Pit Lord

How humans handle it: Exploding/Fire Arrow users carry a second element or magic damage: **Cold Arrow/Ice Arrow** at 1 point each, **Magic Arrow** (what our path buys), or lightning javelin on a weapon swap; they use a merc that does physical/aura damage, and they skip immune packs (V1 "skip or Lower Resist wand"). Fire-immune Hell packs: Act 1-3 each have 10-16 fire-immune regular types, so about 1 in 4 monster types in Hell can't be burned (T). Our line fixes it only with Magic Arrow (low), the merc and skipping, so **in Hell a fire bow Amazon is not recommended without a respec** (U).

## 8. Strengths, weaknesses, deaths, by act and difficulty

- Normal Act 1 (clvl 1-18): strong. Ranged kiting, Fire Arrow one-shots early packs; weaknesses: low mana pool and the ammo/potion economy. Typical deaths (U, community knowledge, not project logs): Blood Raven's cold arrows, Andariel's poison clouds (Normal has 0 penalty but no poison resist yet), Countess's fire bolts.
- Normal Act 2-3: Duriel's cold/stun, Sand Maggot Lair and the Arcane Sanctuary corridors, and Jungle flayers/Spider Cavern for Act 3 (U). The human plan uses a Staff of Teleportation from clvl 20-23 (V1), which the bot cannot swap into.
- Normal Act 4-5: Diablo's lightning and fire nova, then crowded melee in Act 5 (Barbarian soldiers, Frozen Tundra) once the merc is dead (U).
- Nightmare: the resist floor -40 turns every Hell-fire, poison and cold hit up; common deaths are Mephisto's lightning, Fire/Lightning enchanted uniques, Izual, and cursed Doom Knights (T: Act 4 NM has 1 fire-immune type). 
- Hell: Fire/lightning immunes and enchanted packs; the Amazon's life pool is moderate (3 per Vit, T).
- Typical bot deaths (U, predicted from the build, not seen in logs): out of arrows or mana mid-pack, merc dead then melee packs, Duriel's cold, fire-enchanted uniques.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | fit for a no-respec bot |
|---|---|---|---|---|
| Javelin lightning (Power/Charged Strike -> Lightning Fury) | Jab/Power Strike, Poison Javelin, Lightning Bolt, Plague Javelin at 18, Lightning Fury at 30 (T prereqs) | strongest endgame clear (V1 Maxroll Lightning Fury), uses Titan's Revenge | lightning-immune types exist in every act in Hell (T: 8-15 per act), mana heavy, javelin supply | needs Plague/Charged chain points (early skills trapped) |
| Bow fire (this file) | Fire Arrow, Multiple Shot 1, Exploding Arrow | one targeted skill, easy aim, fine to NM | fire immunity (T: 10-16 types per Hell act), arrow/mana economy | best fit to 40 |
| Bow physical/multi (Multiple Shot -> Strafe -> Guided) | Multiple Shot, Guided Arrow, Strafe at 24 (T) | physical immune types are rare (T: 0 in NM, 0-4 per Hell act), cheap in mana at low skill | gear-hungry (IAS, Windforce/ladder bows), Strafe is hold-to-cast and hits 5 targets | good Hell fallback if the gear exists |

## 10. When is the character "ready" (bot checks, all U unless tagged)

Check all of these in the Normal/NM/Hell hop tables; numbers are judgement from the first-pass gates plus this file (T for resist penalties).

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 5-6, Fire Arrow >= 3, 4 health + 1 mana potions, 1 TP tome | 3+ stacks of arrows, belt with 4 columns |
| Andariel / Act 2 | clvl 14-16, life >= 220, Exploding Arrow 1 (clvl 12), Magic Arrow 1, carry antidote potions (U) | merc hired (Act 2 Desert, Defiance/Prayer) |
| Duriel / Act 3 | clvl 20-22, life >= 350, cold resist >= 20 (Rhyme/gem) | staff of tele not needed |
| Mephisto / Act 4 | clvl 26-28, life >= 450, fire/cold/lightning >= 40 each | Exploding Arrow >= 12 |
| Diablo / Act 5 | clvl 30-32, life >= 550 | all res >= 50, Izual done |
| Baal | clvl 33-36, life >= 600, +60 fire/cold/ltng | Rite of Passage reward at clvl >= 20 (V AS rewards page) |
| Enter Nightmare | clvl >= 38-40 (V AS: Act 5 Normal exp is good until clvl 40), raw res >= 60 each (+20 after -40, T), life >= 640 (T path value at clvl 40 before gear is 638), merc alive, 8 of the first 12 quest points done | gold >= price of revive (15,000 at Hell merc, T) |
| Enter Hell | clvl >= 64-68 (project gate 64, V `docs/progression/difficulty.md`), raw res >= 125 each, life >= 1500, a second damage element (Hell has 54 fire-immune regular spawn types across the five acts, 83 fire-immune entries in `monstats.tsv`, T) | mana potions + rejuvs, Act 2 merc alive (Hell offense aura is Blessed Aim) |

If a check fails, the bot should level in the previous act's Hell/NM areas or farm resists instead of hopping (V `docs/progression/difficulty.md` section 3).

## 11. Open questions

1. Are ammo stacks bought by the bot? Quiver size and arrow price (U); read from `misc.tsv`.
2. Valkyrie/Decoy are not in the path; bot survival is by merc and kiting only (U).
3. The Weapon swap does not exist in the bot (`docs/builds-endgame.md`), so Teleport/Lower Resist staff plans are manual-only.

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).


## BUILDV2 implementation note (2026-10-30, fresh Amazon)
Bow line stays the default (`amazon_bow_normal30`, edited in place: Multiple Shot moved from clvl 11 to 7, with the Den point it lands at clvl 6-7; no owned skill is refunded). `amazon_javelin_normal30` is the non-default variant. A bow-path Amazon on the starting Javelin makes a town trip at 150 gold (`bow` trip reason, internal/town/ammo.go `wantsBow`); the ammo chore runs before the potion shop (`chores.go`) and buys Short Bow + one Arrows stack (U: whether 150 gold covers both, the quiver price was not read). Until the bow is worn plain Attack is used.

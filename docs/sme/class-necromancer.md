# SME class file: Necromancer (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companions: `builds-leveling.md`, `builds-endgame-casters.md`, `builds-gear.md`, `builds-open-questions.md`, `docs/classes/necromancer.md`, `docs/summon.md`. Feeds an unattended bot with no respec (`docs/leveler.md`).
Tags: **V** fetched or computed from the install's tables (URL or "T"), **V1** carried from first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary, **U** memory or judgement. D2R-only facts are marked "D2R".

## 0. Verdict

The Summoner (Raise Skeleton + Skeleton Mastery + Skeletal Mages, with a Clay Golem and Amplify Damage) is the safest necromancer for an unattended bot: minions tank and kill, the mana bill is a one-time cost per summon (Raise Skeleton costs 5 + skill level mana, V AB), and it needs no aiming. Its two known weaknesses are (1) **summons do only 50% damage to act bosses in Normal, 35% in Nightmare, 25% in Hell** (V AS difficulty page), so boss kills slow down badly in later difficulties, and (2) minions die to uniques and need re-summoning. Human players use the bone line (Corpse Explosion -> Bone Spear/Prison) for bosses and respec into it at clvl 26 (V1 Maxroll); the bot cannot, so the path below keeps only the summons and one mark for emergencies.

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 15, Dex 25, Int 25, Vit 15 |
| Life | +1.5 per clvl, +2 per Vit; Mana +2 per clvl, +2 per Energy; Stamina +1/+1 |
| Starting life/mana (U) | about 45 / 25 |
| Start skill | Raise Skeleton (T `StartSkill`) |
| Starting equipment | Wand, 4 minor healing potions, TP and ID scrolls (T codes wnd, hp1 x4, tsc, isc) |
| Block / AR | block 20, ToHitFactor -10 (never melee) |

Points: 1 per clvl from 2 plus 4 per difficulty; 47 at clvl 40 on our timeline. Stat points 5 per clvl.

## 2. Skills, synergies, per-level numbers (T)

Added damage per hit (/s = per second per source). Mana at skill level 1/10/20. Summon skills have no damage column; their numbers are below the table.

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Amplify Damage | 1 | - | 4/4/4 | - | - |
| Teeth | 1 | - | 3/7.5/12.5 | 2-4 / 11-14 / 23-31 mag | +15%/lvl: Bone Wall, Bone Prison, Bone Spear, Bone Spirit |
| Bone Armor | 1 | - | 11/20/30 | - | - |
| Skeleton Mastery | 1 | Raise Skeleton | - | - | - |
| Raise Skeleton | 1 | - | 6/15/25 | - | - |
| Dim Vision | 6 | - | 9/9/9 | - | - |
| Weaken | 6 | Amplify Damage | 4/4/4 | - | - |
| Poison Dagger | 6 | - | 3/5.2/7.8 | poison, per-frame units (not resolved) | +20%/lvl: Poison Explosion, Poison Nova |
| Corpse Explosion | 6 | Teeth | 15/24/34 | - | - |
| Clay Golem | 6 | - | 15/42/72 | - | - |
| Iron Maiden | 12 | Amplify Damage | 5/5/5 | - | - |
| Terror | 12 | Weaken | 7/7/7 | - | - |
| Bone Wall | 12 | Bone Armor | 17/17/17 | - | - |
| Golem Mastery | 12 | Clay Golem | - | - | - |
| Raise Skeletal Mage | 12 | Raise Skeleton | 8/17/27 | - | - |
| Confuse | 18 | Dim Vision | 13/13/13 | - | - |
| Life Tap | 18 | Iron Maiden | 9/9/9 | - | - |
| Poison Explosion | 18 | Poison Dagger, Corpse Explosion | 8/8/8 | poison, per-frame units (not resolved) | +15%/lvl: Poison Dagger, Poison Nova |
| Bone Spear | 18 | Corpse Explosion | 7/9.2/11.8 | 16-24 / 90-98 / 192-204 mag | +7%/lvl: Bone Wall, Bone Prison, Teeth, Bone Spirit |
| Blood Golem | 18 | Clay Golem | 25/61/101 | - | - |
| Attract | 24 | Confuse | 17/17/17 | - | - |
| Decrepify | 24 | Terror | 11/11/11 | - | - |
| Bone Prison | 24 | Bone Wall, Bone Spear | 27/18/8 | - | - |
| Summon Resist | 24 | Golem Mastery | 44/17/1 | - | - |
| Iron Golem | 24 | Blood Golem | 35/35/35 | - | - |
| Lower Resist | 30 | Life Tap, Decrepify | 22/22/22 | - | - |
| Poison Nova | 30 | Poison Explosion | 20/20/20 | poison, per-frame units (not resolved) | +10%/lvl: Poison Dagger, Poison Explosion |
| Bone Spirit | 30 | Bone Spear | 12/16.5/21.5 | 20-30 / 166-185 / 340-369 mag | +6%/lvl: Bone Wall, Bone Prison, Teeth, Bone Spear |
| Fire Golem | 30 | Iron Golem | 50/140/240 | 10-27 / 93-119 / 197-233 fire | - |
| Revive | 30 | Raise Skeletal Mage, Iron Golem | 45/45/45 | - | - |

Summoner formulas (V AB `https://theamazonbasin.com/wiki/index.php?title=Raise_Skeleton`, `Skeleton_Mastery`, `Clay_Golem`; our table agrees on mana 5 + slvl):
- Skeletons raised: slvl if slvl < 4, else 2 + floor(slvl/3). Each skeleton: min damage 1+2*SM (levels 1-8), slvl-7+2*SM (9-16), 2*slvl-23+2*SM (17-22), 3*slvl-45+2*SM (23-28), 4*slvl-73+2*SM (29+), max one point more; base life 21+8*SM Normal, 30+8*SM Nightmare, 42+8*SM Hell, and +50*slvl-150 % life from the Raise Skeleton level when slvl >= 4 (applied at summon time).
- Skeleton Mastery gives each skeleton and mage +8 life and skeleton +2 damage per level; revived monsters +5% life and +10% damage per level (V AB). Bonuses are fixed at summon time.
- Clay Golem: mana 12 + 3*slvl, life 100*(65+35*slvl)/100 in Normal (175 NM, 275 Hell multiplier), damage +35% per level, slows target up to 75%; **only one golem at a time** (V AB).

| Raise Skeleton level | skeletons raised | min-max phys damage each (with Skeleton Mastery = same level) | base life before +% (Normal / NM / Hell; SM level = same) | +% life (skill) |
|---|---|---|---|---|
| 1 | 1 | 3-4 | 29 / 38 / 50 | +0% |
| 4 | 3 | 9-10 | 53 / 62 / 74 | +50% |
| 8 | 4 | 17-18 | 85 / 94 / 106 | +250% |
| 10 | 5 | 23-24 | 101 / 110 / 122 | +350% |
| 12 | 6 | 29-30 | 117 / 126 / 138 | +450% |
| 15 | 7 | 38-39 | 141 / 150 / 162 | +600% |
| 18 | 8 | 49-50 | 165 / 174 / 186 | +750% |
| 20 | 8 | 57-58 | 181 / 190 / 202 | +850% |

Notes:
- Raise Skeletal Mage (clvl 12) uses the same Skeleton Mastery life bonus; the mage's element is random (cold, fire, lightning, poison) per mage (S). Mages shoot from range and are the damage dealers in crowds.
- Corpse Explosion (clvl 6, prereq Teeth) damages in an area based on corpse life (S) and needs corpses; Bone Spear (clvl 18, prereq Corpse Explosion) has +7% per level of Teeth/Bone Wall/Bone Prison/Bone Spirit (T, V AB); Bone Armor absorbs damage for mana (S).
- Curses: Amplify Damage (clvl 1) doubles physical damage taken (S) and, in Hell, curse AI effects are cut 3/4 (V AS), so Dim Vision/Confuse/Attract are weak in Hell; Amplify itself is not an AI curse (U). Lower Resist at clvl 30 (needs Life Tap + Decrepify) reduces all resists (U) and is out of reach on this path.

## 3. Mana, potions, early game problems

| clvl | Vit points added | Energy points added | life (no gear) | mana (no gear) |
|---|---|---|---|---|
| 10 | 42 | 3 | 142 | 49 |
| 20 | 67 | 3 | 207 | 69 |
| 30 | 117 | 3 | 322 | 89 |
| 40 | 167 | 3 | 437 | 109 |

- Minions need no mana to fight; the pool only buys summons: Raise Skeleton at level 15 costs 20 mana, a mage at level 12 costs about 19 (T table: 8/17/27 at levels 1/10/20), a golem 15-30. Re-summon after each pack loss; potions are mostly for health. Energy 2 points at clvl 2 (the Maxroll stat plan, V1) is enough.
- Early life is the binding resource: 45 start life, only +1.5 per clvl (T). Skeletons at level 1-3 raise 1-3 units with 29-53 life (table above) so the player dies first in Act 1 Normal if skeletons are not between them and monsters (U). Use Clay Golem (1 point at clvl 6) as the extra target for bosses.
- If potions run out: do not resummon, go to town; skeletons regenerate only 4 damage regen (V AB) so they die. The bot's town trip must come before the first hostile pack when potion count < 2 (project `kit_clvl`, V `docs/bossgate.md`).

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil at clvl 5, Radament 15, Izual 24, NM Den/Radament/Izual at 30/34/38 (U). Machine-validated against `skills.tsv` (reqlevel and prerequisites OK, T). Final allocation at clvl 40: Raise Skeleton 15, Skeleton Mastery 15, Raise Skeletal Mage 12, Bone Armor 3, Clay Golem 1, Amplify Damage 1. The right column shows the three summon skills' levels and mana.

| pt | at | skill point | stat points (5/clvl) | Raise Skeleton / Skeleton Mastery / Raise Skeletal Mage |
|---|---|---|---|---|
| 1 | clvl 2 | Raise Skeleton | 3 Ene, 2 Vit | Raise Skeleton 1 mana 6.0 |
| 2 | clvl 3 | Skeleton Mastery | 5 Vit | Raise Skeleton 1 mana 6.0; Skeleton Mastery 1 |
| 3 | clvl 4 | Raise Skeleton | 5 Vit | Raise Skeleton 2 mana 7.0; Skeleton Mastery 1 |
| 4 | clvl 5 | Skeleton Mastery | 5 Vit | Raise Skeleton 2 mana 7.0; Skeleton Mastery 2 |
| 5 | Den of Evil N (clvl ~5) | Raise Skeleton | - | Raise Skeleton 3 mana 8.0; Skeleton Mastery 2 |
| 6 | clvl 6 | Clay Golem | 5 Vit | Raise Skeleton 3 mana 8.0; Skeleton Mastery 2 |
| 7 | clvl 7 | Skeleton Mastery | 5 Vit | Raise Skeleton 3 mana 8.0; Skeleton Mastery 3 |
| 8 | clvl 8 | Raise Skeleton | 5 Vit | Raise Skeleton 4 mana 9.0; Skeleton Mastery 3 |
| 9 | clvl 9 | Skeleton Mastery | 5 Vit | Raise Skeleton 4 mana 9.0; Skeleton Mastery 4 |
| 10 | clvl 10 | Raise Skeleton | 5 Vit | Raise Skeleton 5 mana 10.0; Skeleton Mastery 4 |
| 11 | clvl 11 | Skeleton Mastery | 5 Vit | Raise Skeleton 5 mana 10.0; Skeleton Mastery 5 |
| 12 | clvl 12 | Raise Skeletal Mage | 5 Str | Raise Skeleton 5 mana 10.0; Skeleton Mastery 5; Raise Skeletal Mage 1 mana 8.0 |
| 13 | clvl 13 | Amplify Damage | 5 Str | Raise Skeleton 5 mana 10.0; Skeleton Mastery 5; Raise Skeletal Mage 1 mana 8.0 |
| 14 | clvl 14 | Bone Armor | 5 Str | Raise Skeleton 5 mana 10.0; Skeleton Mastery 5; Raise Skeletal Mage 1 mana 8.0 |
| 15 | clvl 15 | Bone Armor | 5 Str | Raise Skeleton 5 mana 10.0; Skeleton Mastery 5; Raise Skeletal Mage 1 mana 8.0 |
| 16 | Radament N (clvl ~15) | Bone Armor | - | Raise Skeleton 5 mana 10.0; Skeleton Mastery 5; Raise Skeletal Mage 1 mana 8.0 |
| 17 | clvl 16 | Raise Skeleton | 5 Str | Raise Skeleton 6 mana 11.0; Skeleton Mastery 5; Raise Skeletal Mage 1 mana 8.0 |
| 18 | clvl 17 | Skeleton Mastery | 5 Vit | Raise Skeleton 6 mana 11.0; Skeleton Mastery 6; Raise Skeletal Mage 1 mana 8.0 |
| 19 | clvl 18 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 6 mana 11.0; Skeleton Mastery 6; Raise Skeletal Mage 2 mana 9.0 |
| 20 | clvl 19 | Raise Skeleton | 5 Vit | Raise Skeleton 7 mana 12.0; Skeleton Mastery 6; Raise Skeletal Mage 2 mana 9.0 |
| 21 | clvl 20 | Skeleton Mastery | 5 Vit | Raise Skeleton 7 mana 12.0; Skeleton Mastery 7; Raise Skeletal Mage 2 mana 9.0 |
| 22 | clvl 21 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 7 mana 12.0; Skeleton Mastery 7; Raise Skeletal Mage 3 mana 10.0 |
| 23 | clvl 22 | Raise Skeleton | 5 Vit | Raise Skeleton 8 mana 13.0; Skeleton Mastery 7; Raise Skeletal Mage 3 mana 10.0 |
| 24 | clvl 23 | Skeleton Mastery | 5 Vit | Raise Skeleton 8 mana 13.0; Skeleton Mastery 8; Raise Skeletal Mage 3 mana 10.0 |
| 25 | clvl 24 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 8 mana 13.0; Skeleton Mastery 8; Raise Skeletal Mage 4 mana 11.0 |
| 26 | Izual N (clvl ~24) | Raise Skeleton | - | Raise Skeleton 9 mana 14.0; Skeleton Mastery 8; Raise Skeletal Mage 4 mana 11.0 |
| 27 | Izual N 2 (clvl ~24) | Skeleton Mastery | - | Raise Skeleton 9 mana 14.0; Skeleton Mastery 9; Raise Skeletal Mage 4 mana 11.0 |
| 28 | clvl 25 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 9 mana 14.0; Skeleton Mastery 9; Raise Skeletal Mage 5 mana 12.0 |
| 29 | clvl 26 | Raise Skeleton | 5 Vit | Raise Skeleton 10 mana 15.0; Skeleton Mastery 9; Raise Skeletal Mage 5 mana 12.0 |
| 30 | clvl 27 | Skeleton Mastery | 5 Vit | Raise Skeleton 10 mana 15.0; Skeleton Mastery 10; Raise Skeletal Mage 5 mana 12.0 |
| 31 | clvl 28 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 10 mana 15.0; Skeleton Mastery 10; Raise Skeletal Mage 6 mana 13.0 |
| 32 | clvl 29 | Raise Skeleton | 5 Vit | Raise Skeleton 11 mana 16.0; Skeleton Mastery 10; Raise Skeletal Mage 6 mana 13.0 |
| 33 | clvl 30 | Skeleton Mastery | 5 Vit | Raise Skeleton 11 mana 16.0; Skeleton Mastery 11; Raise Skeletal Mage 6 mana 13.0 |
| 34 | Den of Evil NM (clvl ~30) | Raise Skeletal Mage | - | Raise Skeleton 11 mana 16.0; Skeleton Mastery 11; Raise Skeletal Mage 7 mana 14.0 |
| 35 | clvl 31 | Raise Skeleton | 5 Vit | Raise Skeleton 12 mana 17.0; Skeleton Mastery 11; Raise Skeletal Mage 7 mana 14.0 |
| 36 | clvl 32 | Skeleton Mastery | 5 Vit | Raise Skeleton 12 mana 17.0; Skeleton Mastery 12; Raise Skeletal Mage 7 mana 14.0 |
| 37 | clvl 33 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 12 mana 17.0; Skeleton Mastery 12; Raise Skeletal Mage 8 mana 15.0 |
| 38 | clvl 34 | Raise Skeleton | 5 Vit | Raise Skeleton 13 mana 18.0; Skeleton Mastery 12; Raise Skeletal Mage 8 mana 15.0 |
| 39 | Radament NM (clvl ~34) | Skeleton Mastery | - | Raise Skeleton 13 mana 18.0; Skeleton Mastery 13; Raise Skeletal Mage 8 mana 15.0 |
| 40 | clvl 35 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 13 mana 18.0; Skeleton Mastery 13; Raise Skeletal Mage 9 mana 16.0 |
| 41 | clvl 36 | Raise Skeleton | 5 Vit | Raise Skeleton 14 mana 19.0; Skeleton Mastery 13; Raise Skeletal Mage 9 mana 16.0 |
| 42 | clvl 37 | Skeleton Mastery | 5 Vit | Raise Skeleton 14 mana 19.0; Skeleton Mastery 14; Raise Skeletal Mage 9 mana 16.0 |
| 43 | clvl 38 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 14 mana 19.0; Skeleton Mastery 14; Raise Skeletal Mage 10 mana 17.0 |
| 44 | Izual NM (clvl ~38) | Raise Skeleton | - | Raise Skeleton 15 mana 20.0; Skeleton Mastery 14; Raise Skeletal Mage 10 mana 17.0 |
| 45 | Izual NM 2 (clvl ~38) | Skeleton Mastery | - | Raise Skeleton 15 mana 20.0; Skeleton Mastery 15; Raise Skeletal Mage 10 mana 17.0 |
| 46 | clvl 39 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 15 mana 20.0; Skeleton Mastery 15; Raise Skeletal Mage 11 mana 18.0 |
| 47 | clvl 40 | Raise Skeletal Mage | 5 Vit | Raise Skeleton 15 mana 20.0; Skeleton Mastery 15; Raise Skeletal Mage 12 mana 19.0 |

Stat plan: clvl 2 three Energy and two Vit, Vit to clvl 16 interleaved with Strength 5 per level to wear light armor (clvl 12-16), then Vit (U; Maxroll: Energy early then Vit, Str 22 around clvl 17-21, V1).
Why: skeletons and Skeleton Mastery together to ~15 each by clvl 40 (T formula above: count 7, damage 38-39 each at level 15), Skeletal Mage 12 for ranged damage, Clay Golem 1 and Amplify Damage 1 as utilities, Bone Armor 3 for survivability. No Bone Spear, Corpse Explosion or Revive on this path (Revive needs Iron Golem + Skeletal Mage, c30, six points of prerequisites, U).

## 5. Mercenary

- Act 2 Desert merc (V AS/T): Prayer (heal aura) or Defiance in Normal and Hell; Thorns/Holy Freeze/Might in Nightmare. For the Summoner the aura is shared with minions (V AS: Desert mercs "share their auras with all of the party, including Minions"), so Defiance/Prayer is the best bot choice; Might in NM raises minion damage too. Hire cost: 350 gold (Normal), 7,900 (NM), 15,000 (Hell) (T).
- Act 1 Rogue is optional; Act 5 Barbarian has the most life but only Bash/Stun (T/V AS).
- Equipment: spears/polearms for Act 2; no Infinity offline (ladder, V `builds-open-questions.md` Q3).

## 6. Gear per act on a budget

| stage | weapon | armor / helm / shield | rest | notes |
|---|---|---|---|---|
| Normal Act 1-2 | start Wand; a magic wand with +Skeleton Mastery/+skills (V AB: +1-3 SM can roll on ilvl<25 wands/heads) | cheap armor with sockets; Stealth (Tal+Eth, T) at clvl 17 (V1) | belt | Leaf is a sorceress staff; skip |
| Normal Act 3-5 | White (Dol+Io wand, T) once a Dol rune is found: +3 Poison&Bone tab, +3 Bone Armor, +2 Bone Spear, +4 Skeleton Mastery, 20% FCR, +13 mana (T `runes.tsv`, AB shows +4 SM) | Lore helm (Ort+Sol: +1 all skills, 2 light, T), necromancer shrunken head/shield with +skills (U) | belt + boots | the runeword Bone (Sol+Um+Um, body armor, T) gives necromancer bonuses (S) |
| Nightmare | White wand; Arm of King Leoric/Undead Crown give +3 SM (V AB table) | Chains of Honor (+2 all skills, V open-questions Q2) or Enigma later; Stone not needed (low FHR) | Rhyme shield | raw res 60 each by NM entry (V AS -40) |
| Hell | White, Heart of the Oak (staff, +3 skills, T) | Enigma (+2 skills, Teleport), Harlequin Crest | | resist target raw 125 |

## 7. Resists and immunities

Targets: Normal 0 needed; Nightmare entry raw 60 fire/cold/lightning (+20 after -40); Hell raw 125 (+25 after -100), cap 175 (V AS). Poison resist matters more for a poison-less Necromancer in Act 3 (U).

Immunity landscape (T, distinct spawn types immune; NM / Hell):
| act | distinct spawn types | physical imm NM / Hell | magic imm NM / Hell | fire imm NM / Hell | poison imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 0 | 0 / 16 | 0 / 6 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 0 / 3 | 1 / 13 | 0 / 11 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 2 | 0 / 10 | 0 / 11 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 0 / 0 | 1 / 4 | 1 / 4 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 0 / 1 | 2 / 11 | 14 / 6 |

- Hell magic-immune spawn types by act:
  - Act 2: Plague Bearer, Guardian, Unraveler
  - Act 3: Horadrim Ancient, Wailing Beast
  - Act 5: Pit Lord
- Hell phys-immune spawn types by act:
  - Act 1: Ghost, Wraith
  - Act 2: Itchies, Black Locusts, Apparition, Specter
  - Act 3: Hell Swarm, Dark Shape
  - Act 5: Hell Temptress, Blood Lord, Stygian Fury, Hell Swarm

How humans handle it: physical (skeleton) immunes are rare (T above: 2-4 per Hell act, 0 in NM) and are cured by **Amplify Damage** (mages and golems) or by the bone/poison/curse skills; magic immunes (Unravelers, Guardians, Horadrim Ancients, Pit Lords, Plague Bearers, Wailing Beasts, T) stop Bone Spear but not skeletons or mages. A Summoner with 12 Skeletal Mages (random elements) naturally covers elemental immunes (S). That is why the Summoner is the most immunity-robust class for a bot (U), with the cost of the boss damage cut above.

## 8. Strengths, weaknesses, deaths

- Normal Act 1-2: slow first levels (1-3 skeletons with 29-53 life, T), strong after skeletons reach level 8+ (4 units, 17-18 damage each, T). Weak vs Blood Raven's fire arrows if skeletons are not between you and her; Andariel's poison cloud kills the whole army.
- Normal Act 3: jungle fire/lightning casters and Sexton kill skeletons quickly; golem helps.
- Normal Act 4-5: Diablo's fire nova/lightning; Baal waves with lightning casters; **boss damage by summons is 50% in Normal** (V AS), so Mephisto/Diablo/Baal take 2x as long: the bot must be able to survive minion death (potions, TP).
- Nightmare: summons deal 35% to bosses (V AS); Act 4 NM has 1 fire/lightning/poison-immune type each (T), so fine; resist -40 plus 130-250 life (T) is the danger; frequent merc revives (V AS).
- Hell: boss damage 25% (V AS) makes boss kills very long for the Summoner (U: need Corpse Explosion/Bone Spear in your own hands); Hell resists -100.
- Typical bot deaths (U, predicted): standing in melee with minions dead; Andariel/Duriel boss fights at 25-50% minion damage; fire/lightning enchanted packs ignoring minions.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | no-respec fit |
|---|---|---|---|---|
| Summoner (this file) | Raise Skeleton, Skeleton Mastery, Skeletal Mage, Clay/Blood/Iron Golem, Revive (T prereqs) | safest, immunity-robust, mana light | bosses take 2-4x longer (V AS cuts), slow leveling to clvl ~15 | best |
| Bone Spear / Prison (V1 Maxroll) | Teeth, Corpse Explosion (prereq), Bone Spear c18, Bone Prison c24, Bone Spirit c30 | highest boss DPS, own damage not cut | magic immunes (T: 0 NM, 1-6 types per Hell act), mana, fragile | needs respec for the early game (V1 respec at 26) |
| Poison Nova | Poison Dagger, Poison Explosion c18, Poison Nova c30 | fast clear | poison immunes (T: 6-14 types per act in NM/Hell), needs Death's Web-type gear (V1) | poor |

## 10. When is the character "ready" (bot checks; U unless tagged)

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 7-8, Raise Skeleton >= 3, Skeleton Mastery >= 2 (3 skeletons), Clay Golem 1 at clvl 6 | 4 health potions, mana for 3 summons |
| Andariel | clvl 16-18, Raise Skeleton + SM 5 each, mage 1+, life >= 150, 6 health potions | Defiance/Prayer merc |
| Duriel / Act 3 | clvl 20-22, life >= 200, minions alive at pack start | cold resist >= 20 |
| Mephisto | clvl 24-26, life >= 220, RS 9 (5 skeletons, V AB formula) | merc alive |
| Diablo / Baal | clvl 28-34, life >= 260-300, Skeletal Mage 8 | all res >= 40-60 |
| Enter Nightmare | clvl >= 38-40, raw res >= 60, life >= 330 (T path value 40), Skeleton Mastery >= 14 | merc alive, 6 health potions |
| Enter Hell | clvl >= 64 (project gate V), raw res >= 125, life >= 900, a boss-damage plan (own damage skill) | summons cut to 25% on bosses (V AS) |

## 11. Open questions

1. Skeletal Mage damage/life and element probabilities (U): not in the extract columns read; read from `skills.tsv`/`monstats.tsv` rows `nmage*` (not yet).
2. Does Amplify Damage remove full physical immunity (U)?
3. Starting life/mana (U).

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).


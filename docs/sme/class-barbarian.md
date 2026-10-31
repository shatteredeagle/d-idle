# SME class file: Barbarian (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companions: `builds-leveling.md`, `builds-endgame-melee.md`, `builds-gear.md`, `builds-open-questions.md`, `docs/classes/barbarian.md`. Feeds an unattended bot with no respec (`docs/leveler.md`).
Tags: **V** fetched or computed from the install's tables (URL or "T"), **V1** carried from first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary, **U** memory or judgement. D2R-only facts are marked "D2R".

## 0. Verdict

The Barbarian has the biggest life pool in the game (+4 per Vit point, +2 per clvl, T) and almost no mana needs, so he is the easiest class to keep alive without potions of mana. His problems for a bot: melee positioning, stamina, **physical immune packs in Hell** (T: 2-4 regular types per act, none in Nightmare) and **dual wielding**, which needs two one-handed weapons equipped and a gear manager that can handle that. The no-respec path below is the **Frenzy / Double Swing sword barbarian**: Sword Mastery first, Double Swing from clvl 6 (mana free from skill level 9), Frenzy from clvl 24 (+85+5*slvl% damage, V AB), and Battle Orders (+32+3*slvl% life/mana/stamina to the party, V AB) for survivability.

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 30, Dex 20, Int 10, Vit 25 |
| Life | +2 per clvl, +4 per Vit (best); Mana +1 per clvl, +1 per Energy; Stamina +1/+1 |
| Starting life/mana (U) | about 55 / 10 |
| Starting equipment | Hand Axe, Buckler, 4 minor healing potions, TP and ID scrolls (T codes hax, buc, hp1 x4, tsc, isc) |
| Block / AR | block 25, ToHitFactor 20 per Dex |

Points: 1 per clvl from 2 plus 4 per difficulty; 47 at clvl 40 on our timeline. Stat points 5 per clvl.

## 2. Skills, synergies, per-level numbers (T)

Barbarian skills mostly have no added-damage column (they are weapon based); see the formulas below the table. Mana shown at skill level 1/10/20.

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Bash | 1 | - | 2/2/2 | - | - |
| Sword Mastery | 1 | - | - | - | - |
| Axe Mastery | 1 | - | - | - | - |
| Mace Mastery | 1 | - | - | - | - |
| Howl | 1 | - | 4/4/4 | - | - |
| Find Potion | 1 | - | 2/2/2 | - | - |
| Leap | 6 | - | 2/2/2 | - | - |
| Double Swing | 6 | Bash | 1/0/0 | - | - |
| Pole Arm Mastery | 6 | - | - | - | - |
| Throwing Mastery | 6 | - | - | - | - |
| Spear Mastery | 6 | - | - | - | - |
| Taunt | 6 | Howl | 3/3/3 | - | - |
| Shout | 6 | Howl | 6/6/6 | - | - |
| Stun | 12 | Bash | 2/2/2 | - | - |
| Double Throw | 12 | Double Swing | 1/1/1 | - | - |
| Increased Stamina | 12 | - | - | - | - |
| Find Item | 12 | Find Potion | 7/7/7 | - | - |
| Leap Attack | 18 | Leap | 9/9/9 | - | - |
| Concentrate | 18 | Stun | 2/2/2 | - | - |
| Iron Skin | 18 | - | - | - | - |
| Battle Cry | 18 | Taunt | 5/5/5 | - | - |
| Frenzy | 24 | Double Throw | 1.5/1.5/1.5 | - | - |
| Increased Speed | 24 | Increased Stamina | - | - | - |
| Battle Orders | 24 | Shout | 7/7/7 | - | - |
| Grim Ward | 24 | Find Item | 4/4/4 | - | - |
| Whirlwind | 30 | Leap Attack, Concentrate | 12.5/17/22 | - | - |
| Berserk | 30 | Concentrate | 4/4/4 | - | - |
| Natural Resistance | 30 | Iron Skin | - | - | - |
| War Cry | 30 | Battle Cry, Battle Orders | 10/19/29 | - | +6%/lvl: Howl, Taunt, Battle Cry |
| Battle Command | 30 | Battle Orders | 11/11/11 | - | - |

Formulas (V AB `https://theamazonbasin.com/wiki/index.php?title=<Skill>`, our mana column agrees):
- **Frenzy** (clvl 24, prereq Double Throw): 100% weapon damage, +85+5*slvl% damage, +8% damage per Double Swing and Taunt level, +93+7*slvl% attack rating, +% magic damage = Berserk level, each hit gives a stacking attack-speed buff (first hit +7% up to +50% at the cap) and move speed up to +200%, duration 6 s. Mana 1.5 per weapon, cast once per weapon (so 3 mana per sequence at level 1). At Frenzy 20, Double Swing 8, Taunt 1: damage bonus 85+100 + 8*9 = 257%.
- **Double Swing** (clvl 6, prereq Bash): +10+5*slvl% AR, +10% damage per Bash level, mana 1.125 - 0.125*slvl, **free from skill level 9**; requires two weapons (one target each when several are in range).
- **Sword Mastery**: +20+8*slvl% AR, +23+5*slvl% damage, critical strike chance min(35*(110*slvl/(slvl+6))/100, 35)% (LoD; V AB), only with swords. Axe and Mace Mastery have the same Param table (T Param 28/8/28/5/0/35).
- **Battle Orders** (clvl 24, prereq Shout): mana 7, +32+3*slvl% maximum stamina/life/mana to caster, party, merc and pets, duration 20+10*slvl seconds (V AB). At level 3: +41%. Applies to base life only (levels, Vit, +life items; V AB).
- **Iron Skin** (clvl 18): +20+10*slvl% defense (V AB).
- Whirlwind (clvl 30, prereq Leap Attack + Concentrate) and Berserk (clvl 30, prereq Concentrate) both need Stun and Concentrate points that are not on this path (T prereqs).

## 3. Mana, potions, early game problems

| clvl | Vit points added | Energy points added | life (no gear) | mana (no gear) |
|---|---|---|---|---|
| 10 | 27 | 0 | 181 | 19 |
| 20 | 53 | 0 | 305 | 29 |
| 30 | 91 | 0 | 477 | 39 |
| 40 | 141 | 0 | 697 | 49 |

- Mana is a non-issue: Double Swing is free from level 9 and Frenzy costs 3 per sequence (V AB). Battle Orders costs 7 per cast and must be recast every 20+10*slvl seconds (50 s at level 3, V AB); the bot should recast it when its buff state is gone. Energy points: none.
- Stamina is the Barbarian's resource: buy Stamina potions early (V1 Maxroll Barbarian) and rest in town; Increased Stamina is not on this path (U: Frenzy movement bonus partly compensates).
- Early life at clvl 2-12 is high (55 start, T) and +4 per Vit gives 181 at clvl 10 and 305 at clvl 20 in the table above (T, before gear); deaths are usually from melee in Act 2 (Sand Maggots, Dune Beasts) and Act 3 flayer packs (U).
- If potions run out: do not dive; town and buy; Battle Orders adds 32%+ life on the next cast (V AB), but only if cast before fighting.
- Weapons: dual wield needs two one-handed weapons equipped; the bot must buy or find a second one-hand weapon of the mastery type (swords). Without the second weapon Double Swing/Frenzy become single attacks (S). This is a bot gear requirement, not a skill choice.

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil at clvl 5, Radament 15, Izual 24, NM Den/Radament/Izual at 30/34/38 (U). Machine-validated against `skills.tsv` (reqlevel and prerequisites OK, T). Final allocation at clvl 40: Frenzy 20, Sword Mastery 10, Double Swing 8, Battle Orders 3, Bash 1, Double Throw 1, Howl 1, Taunt 1, Shout 1, Iron Skin 1. The right column shows Double Swing, Sword Mastery, Frenzy and Battle Orders levels with mana.

| pt | at | skill point | stat points (5/clvl) | Double Swing / Sword Mastery / Frenzy / Battle Orders |
|---|---|---|---|---|
| 1 | clvl 2 | Bash | 3 Str, 2 Vit |  |
| 2 | clvl 3 | Sword Mastery | 5 Vit | Sword Mastery 1 |
| 3 | clvl 4 | Sword Mastery | 5 Vit | Sword Mastery 2 |
| 4 | clvl 5 | Sword Mastery | 5 Vit | Sword Mastery 3 |
| 5 | Den of Evil N (clvl ~5) | Sword Mastery | - | Sword Mastery 4 |
| 6 | clvl 6 | Double Swing | 5 Vit | Double Swing 1 mana 1.0; Sword Mastery 4 |
| 7 | clvl 7 | Double Swing | 5 Vit | Double Swing 2 mana 0.9; Sword Mastery 4 |
| 8 | clvl 8 | Double Swing | 3 Str, 2 Dex | Double Swing 3 mana 0.8; Sword Mastery 4 |
| 9 | clvl 9 | Sword Mastery | 3 Str, 2 Dex | Double Swing 3 mana 0.8; Sword Mastery 5 |
| 10 | clvl 10 | Sword Mastery | 3 Str, 2 Dex | Double Swing 3 mana 0.8; Sword Mastery 6 |
| 11 | clvl 11 | Sword Mastery | 5 Vit | Double Swing 3 mana 0.8; Sword Mastery 7 |
| 12 | clvl 12 | Double Throw | 5 Vit | Double Swing 3 mana 0.8; Sword Mastery 7 |
| 13 | clvl 13 | Howl | 3 Str, 2 Vit | Double Swing 3 mana 0.8; Sword Mastery 7 |
| 14 | clvl 14 | Double Swing | 3 Str, 2 Vit | Double Swing 4 mana 0.6; Sword Mastery 7 |
| 15 | clvl 15 | Taunt | 3 Str, 2 Vit | Double Swing 4 mana 0.6; Sword Mastery 7 |
| 16 | Radament N (clvl ~15) | Shout | - | Double Swing 4 mana 0.6; Sword Mastery 7 |
| 17 | clvl 16 | Double Swing | 3 Str, 2 Vit | Double Swing 5 mana 0.5; Sword Mastery 7 |
| 18 | clvl 17 | Double Swing | 3 Str, 2 Vit | Double Swing 6 mana 0.4; Sword Mastery 7 |
| 19 | clvl 18 | Sword Mastery | 3 Str, 2 Vit | Double Swing 6 mana 0.4; Sword Mastery 8 |
| 20 | clvl 19 | Sword Mastery | 3 Str, 2 Vit | Double Swing 6 mana 0.4; Sword Mastery 9 |
| 21 | clvl 20 | Iron Skin | 3 Str, 2 Vit | Double Swing 6 mana 0.4; Sword Mastery 9 |
| 22 | clvl 21 | Double Swing | 3 Str, 2 Vit | Double Swing 7 mana 0.2; Sword Mastery 9 |
| 23 | clvl 22 | Sword Mastery | 3 Str, 2 Vit | Double Swing 7 mana 0.2; Sword Mastery 10 |
| 24 | clvl 23 | Double Swing | 3 Str, 2 Vit | Double Swing 8 mana 0.1; Sword Mastery 10 |
| 25 | clvl 24 | Frenzy | 3 Str, 2 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 1 mana 1.5 |
| 26 | Izual N (clvl ~24) | Frenzy | - | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 2 mana 1.5 |
| 27 | Izual N 2 (clvl ~24) | Frenzy | - | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 3 mana 1.5 |
| 28 | clvl 25 | Battle Orders | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 3 mana 1.5; Battle Orders 1 mana 7.0 |
| 29 | clvl 26 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 4 mana 1.5; Battle Orders 1 mana 7.0 |
| 30 | clvl 27 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 5 mana 1.5; Battle Orders 1 mana 7.0 |
| 31 | clvl 28 | Battle Orders | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 5 mana 1.5; Battle Orders 2 mana 7.0 |
| 32 | clvl 29 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 6 mana 1.5; Battle Orders 2 mana 7.0 |
| 33 | clvl 30 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 7 mana 1.5; Battle Orders 2 mana 7.0 |
| 34 | Den of Evil NM (clvl ~30) | Frenzy | - | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 8 mana 1.5; Battle Orders 2 mana 7.0 |
| 35 | clvl 31 | Battle Orders | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 8 mana 1.5; Battle Orders 3 mana 7.0 |
| 36 | clvl 32 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 9 mana 1.5; Battle Orders 3 mana 7.0 |
| 37 | clvl 33 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 10 mana 1.5; Battle Orders 3 mana 7.0 |
| 38 | clvl 34 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 11 mana 1.5; Battle Orders 3 mana 7.0 |
| 39 | Radament NM (clvl ~34) | Frenzy | - | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 12 mana 1.5; Battle Orders 3 mana 7.0 |
| 40 | clvl 35 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 13 mana 1.5; Battle Orders 3 mana 7.0 |
| 41 | clvl 36 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 14 mana 1.5; Battle Orders 3 mana 7.0 |
| 42 | clvl 37 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 15 mana 1.5; Battle Orders 3 mana 7.0 |
| 43 | clvl 38 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 16 mana 1.5; Battle Orders 3 mana 7.0 |
| 44 | Izual NM (clvl ~38) | Frenzy | - | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 17 mana 1.5; Battle Orders 3 mana 7.0 |
| 45 | Izual NM 2 (clvl ~38) | Frenzy | - | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 18 mana 1.5; Battle Orders 3 mana 7.0 |
| 46 | clvl 39 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 19 mana 1.5; Battle Orders 3 mana 7.0 |
| 47 | clvl 40 | Frenzy | 5 Vit | Double Swing 8 mana 0.1; Sword Mastery 10; Frenzy 20 mana 1.5; Battle Orders 3 mana 7.0 |

Stat plan: Str and Vit early (3 Str, 2 Vit at clvl 2), Vit through clvl 10 with Dex 2 per level at 8-10, 3 Str / 2 Vit from 13 to 24 for armor, then pure Vit (U). Maxroll uses Vit then Dex 40 for Whirlwind (V1), which this path does not take.
Why: Sword Mastery 10 gives +73% damage, +100 AR and 24% crits by level 10 (V AB formulas) and applies to every attack; Double Swing 8 is free of mana and +10+5*slvl% AR (V AB); Frenzy 20 from clvl 24; Battle Orders 3 (+41% life); Iron Skin 1 and Howl/Taunt/Shout/Double Throw 1 each are prerequisites (Taunt also feeds Frenzy +8%/lvl). Humans (V1 Maxroll) go Bash -> Double Swing -> Frenzy, with Leap Attack, Battle Cry and Mace Mastery and a Cow-level respec at 31 into War Cry or Whirlwind; the bot stays on Frenzy.

## 5. Mercenary

- Act 2 Desert merc (V AS/T): Prayer / Thorns / Prayer (Combat), Defiance / Holy Freeze / Defiance, Blessed Aim / Might / Blessed Aim (Offense) by difficulty. Might (NM only) adds damage to a melee Barbarian; Defiance/Prayer in Hell. Best bot pick: Defiance in Normal, Might in NM, Defiance in Hell (U).
- Act 5 Barbarian merc (Bash, Stun; one sword; V AS/T) has the highest life (T: 1,339 at level 80 in NM) but dual-wield is D2R-only; he is not a damage dealer.
- Act 1 Rogue only as a stopgap (V AS).

## 6. Gear per act on a budget

| stage | weapon (two one-handers) | armor / helm | rest | notes |
|---|---|---|---|---|
| Normal Act 1-2 | start Hand Axe; buy swords with +damage or sockets; Malice (Ith+El+Eth, T) / Steel (Tir+El) in 2-3 socket weapons | Stealth (Tal+Eth) at clvl 17 (V1), Nadir/Lore helm | belt from Fara | Charsi sells socketless; Larzuk is Act 5 |
| Normal Act 3-5 | Honor (Amn+El+Ith+Tir+Sol: +160% ed, +1 all skills, +10 Str, T) or King's Grace (Amn+Ral+Thul sword/scepter, T) | Smoke (Nef+Lum) / Lionheart (Hel+Lum+Fal) (T) | resist jewelry, boots with MF/speed | Ral from Act 5 quest (V Q1) |
| Nightmare | Fury (Jah+Gul+Eth: +209% ed, +40% IAS, +33% deadly strike, +6% life steal, T), Kingslayer (Mal+Um+Gul+Fal) if swords | Chains of Honor (+2 skills, +65 res with Um, V Q2) or Stone (+40% FHR, +16 Str, +16 Vit, +220-260% ed; excellent for a melee class, T) | belts, IAS gloves | raw res 60 each at NM entry |
| Hell | Breath of the Dying (6 runes, T), Doom, Eternity (T) | Enigma (+2 skills), Chains of Honor/Stone | Harlequin Crest | raw res 125 each |

Ladder-only skip list (T): Grief (the human Barbarian weapon, V1), Last Wish, Infinity, Fortitude, Obedience, Voice of Reason, Death, Insight, etc. Barbarian offline substitutes are Fury, Kingslayer, Beast (axe/scepter/hammer only), Crescent Moon, Hand of Justice (T).

## 7. Resists and immunities

Targets: Normal 0 needed; Nightmare entry raw 60 fire/cold/lightning (+20 after -40); Hell raw 125 (+25 after -100), cap 175 (V AS). Barbarian has Natural Resistance at clvl 30 (not on path, T prereq Iron Skin); gear is the source.

Immunity landscape (T; NM / Hell types):
| act | distinct spawn types | physical imm NM / Hell | magic imm NM / Hell | fire imm NM / Hell | lightning imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 0 | 0 / 16 | 0 / 14 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 0 / 3 | 1 / 13 | 0 / 11 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 2 | 0 / 10 | 0 / 15 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 0 / 0 | 1 / 4 | 1 / 8 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 0 / 1 | 2 / 11 | 4 / 8 |

- Hell phys-immune spawn types by act:
  - Act 1: Ghost, Wraith
  - Act 2: Itchies, Black Locusts, Apparition, Specter
  - Act 3: Hell Swarm, Dark Shape
  - Act 5: Hell Temptress, Blood Lord, Stygian Fury, Hell Swarm

How humans handle it: physical immune types in Hell: Ghost, Wraith (Act 1), Itchies, Black Locusts, Apparition, Specter (Act 2), Hell Swarm, Dark Shape (Act 3), Hell Temptress, Blood Lord, Stygian Fury, Hell Swarm (Act 5) (T). Barbarians (a) use **Berserk** (magic damage, at clvl 30, needs Concentrate; Frenzy also gets +magic damage equal to Berserk level, V AB) or elemental weapons (Hand of Justice fire, Holy Freeze Doom cold, T), (b) let the merc/summons deal with it, (c) skip those packs. Our no-Berserk path has no answer to physical immunes except a weapon with elemental damage. It is a small share of spawn types (T: 2-4 per Hell act out of 55-57 types) but they include the Act 1-3 ghosts.

## 8. Strengths, weaknesses, deaths by act/difficulty

- Normal Act 1-2 (clvl 1-24): strong; tank with high life; weaknesses: no AoE (V1 gas potions), physical melee positioning; Duriel's cold/stun. Deaths: Blood Raven, Andariel poison, Sand Maggot Lair (U).
- Normal Act 3: flayers and fire/lightning casters; Mephisto's lightning. Act 4: Diablo; Act 5: soldiers and Baal waves; Frenzy from 24 changes the class (+185% damage at level 20 before synergy, V AB).
- Nightmare: -40 resists, but the highest life of any class (T: about 700 at clvl 40 before gear); deaths are lightning/fire enchants, Mephisto, Izual (U).
- Hell: physical immunes, fire/cold/lightning enchants, Hell -100 resists (V AS); stamina; Frenzy's speed and Battle Orders (+life) help.
- Typical bot deaths (U, predicted): fighting without Battle Orders; no second weapon so damage halves; stamina out in a retreat; melee into a pack of physical immune ghosts with no elemental damage.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | no-respec fit |
|---|---|---|---|---|
| Frenzy / Double Swing (this file) | Sword/Axe/Mace Mastery, Double Swing, Frenzy, Battle Orders | fast, huge life, cheap mana, fastest dual-wield melee | physical immunes, two weapons, stamina | best |
| Whirlwind | Leap Attack, Concentrate (Stun->Bash), Whirlwind at 30, Natural Resistance | huge crowd damage | needs Dex 40+, positional play, Berserk for immunes, bot controls hard | poor for a bot |
| War Cry / Berserk (magic) | Battle Cry, Battle Orders, War Cry at 30, Concentrate/Berserk | magic damage ignores physical immunes (V AB: Berserk magic) | single 1h + shield (or 2h), weak early, mana higher | needs respec |

## 10. When is the character "ready" (bot checks; U unless tagged)

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 5-7, Sword Mastery 2+, two one-handers equipped, 4 health potions, 2 Stamina potions | TP tome |
| Andariel | clvl 15-17, life >= 260, Double Swing 3+ | merc, 6 health potions |
| Duriel / Act 3 | clvl 20-22, life >= 360, cold res >= 20, Battle Orders not yet | Iron Skin 1 |
| Mephisto | clvl 26-28, life >= 440, Frenzy 3+ (clvl 24), all res >= 40 | Battle Orders 1 |
| Diablo / Baal | clvl 30-34, life >= 520-600, Frenzy 8+ | all res >= 50-60 |
| Enter Nightmare | clvl >= 38-40, raw res >= 60, life >= 690 (T path value at clvl 40 is 697 before gear), Frenzy >= 18, Battle Orders 3 | 4 health potions, merc alive |
| Enter Hell | clvl >= 64 (project gate V), raw res >= 125, life >= 1500, an elemental weapon or Berserk for physical immunes | Stamina potions, Battle Orders |

## 11. Open questions

1. Does our bot's equip logic support two main-hand weapons (dual wield)? U.
2. Starting life/mana for all classes are U.
3. The Frenzy attack-speed stack in 1.14d (V AB shows D2R duration text separately): exact 1.14d values not independently verified.

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).


## BUILDV2 implementation note (2026-10-30)
The combat profile opens with Bash, then Double Swing, then Attack (Double Swing needs two weapons to be useful, the kit has one; U live check: `combat decision` lines at clvl 2-4 must show Bash).

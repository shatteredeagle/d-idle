# D2 SME KB: breakpoints (FCR / FHR / FBR / IAS) for classic LoD 1.14d

Author: SMEKB-A. Tags: **V** fetched primary/reliable (URL in Sources), **S** secondary/fan/snippet, **U** unverified/memory, **C** computed by
us from a V formula and cross-checked against a V table (script logic reproduced in section 6).

Version note: the animation engine (25 fps, formulas below) is identical in 1.10-1.14d and D2R as far as the sources show. D2R 2.4 changed
only a few edge cases (shapeshift attack speed uses the faster of two equations, S/V M1). The Warlock (RotW, 2026) is out of scope.

## 1. Frame model

- The engine runs 25 frames per second; every action has an integer frame length. Gains only appear when a whole frame drops, hence
  "breakpoints" (V M1, V W1).
- Effective value (diminishing returns), all of FCR/FHR/FBR: `E = floor(X * 120 / (X + 120))` where X is the summed item % (V M1).
  FCR is hard-capped at E=75 (so >200% FCR is useless) (V M1). Same shape used for IAS on items.
- Frames: `Frames = ceil( 256 * Base / floor( AnimSpeed * (K + E) / 100 ) ) - 1`, with K=100 for cast rate and K=50 for hit recovery and block
  (V M1). Lightning/Chain Lightning drop the `-1` (V M1).
- Only the hit-recovery/cast/attack animation lengths matter; they do not change damage per hit.

## 2. Faster Cast Rate (FCR): frames at each breakpoint (C, matches V W1 and V M1 where compared)

FCR% needed to reach each frame count (0% = base). Read left to right: `value%:frames`.

| Class / form | Table |
|---|---|
| Amazon (bow) | 0:19, 7:18, 14:17, 22:16, 32:15, 48:14, 68:13, 99:12, 152:11 |
| Assassin | 0:16, 8:15, 16:14, 27:13, 42:12, 65:11, 102:10, 174:9 |
| Barbarian (Warcry/Find Item) | 0:13, 9:12, 20:11, 37:10, 63:9, 105:8, 200:7 |
| Druid (human) | 0:18, 4:17, 10:16, 19:15, 30:14, 46:13, 68:12, 99:11, 163:10 |
| Werebear | 0:16, 7:15, 15:14, 26:13, 40:12, 63:11, 99:10, 163:9 |
| Werewolf | 0:16, 6:15, 14:14, 26:13, 40:12, 60:11, 95:10, 157:9 |
| Necromancer (human) | 0:15, 9:14, 18:13, 30:12, 48:11, 75:10, 125:9 (see note 1) |
| Necromancer, Trang-Oul's vampire form | 0:23, 6:22, 11:21, 18:20, 24:19, 35:18, 48:17, 65:16, 86:15, 120:14, 180:13 |
| Paladin | 0:15, 9:14, 18:13, 30:12, 48:11, 75:10, 125:9 |
| Sorceress (all spells except below) | 0:13, 9:12, 20:11, 37:10, 63:9, 105:8, 200:7 |
| Sorceress Lightning / Chain Lightning | 0:19, 7:18, 15:17, 23:16, 35:15, 52:14, 78:13, 117:12, 194:11 |
| Act 3 mercenary (Iron Wolf) | 0:17, 8:16, 15:15, 26:14, 39:13, 58:12, 86:11, 138:10 |

Amazon FCR depends on the weapon in hand (V M1 Amazon tab; the Cast Frames column is the total animation, "Action Flag" the frame the skill fires):
- bow: breakpoints 0/7/14/22/32/48/68/99/152 give 19,18,17,16,15,14,13,12,11 frames; action flag 10,10,9,9,8,8,7,7,6.
- crossbow or two-handed swords: 0/7/11/14/22/26/32/48/68/86/99/152/180 give 19,18,18,17,16,16,15,14,13,13,12,11,11 frames; flags 12,12,11,11,11,10,10,9,9,8,8,8,7.
- no weapon: 0/7/13/14/22/29/32/48/56/68/99/113/152 give 19,18,18,17,16,16,15,14,14,13,12,12,11 frames; flags 11,11,10,10,10,9,9,9,8,8,8,7,7.
- two-handed staves, hammers, axes, polearms, spears, 1h knives, javelins: 0/7/10/14/22/23/32/42/48/68/72/99/133/152 give 19,18,18,17,16,16,15,15,14,13,13,12,12,11 frames; flags 13,13,12,12,12,11,11,10,10,10,9,9,8,8.
When the frame count repeats, the breakpoint still moves the action flag one frame earlier, which is what matters for firing cadence.

Notes
1. **Casting base values disagree between sources.** M1 prints bases Druid 16 / Werewolf 16 / Werebear 16 / Necro 15; those give 19-frame Druid
   and different breakpoints than M1's own FCR tables, Wowhead W1 and the classic community tables. Solving for W1's breakpoint lists gives Druid base 15 at
   anim speed 208, Werebear 15 at 228, Werewolf 15 at 229, Necromancer 16 at 256 (C). Those are what the table above uses; W1's own frame column for
   Necromancer reads 16f at 0% but its breakpoints match a 15f start (we show 15f; U which is right in-game, check in the game). Druid/Werebear tables are the
   same in D2R and 1.14d. (S/C)
2. Sorceress cast delay skills (Blizzard 1.8 s, Meteor 1.2, Fire Wall 1.4, Hydra 1.6, Frozen Orb 1.0), Druid Armageddon/Hurricane 6 s, Volcano 4 s, Fissure
   2 s, Molten Boulder 2 s, Firestorm 0.6 s, Paladin Fist of the Heavens 1.0 s, Assassin Shock Web 0.6 s, Blade Sentinel 2 s, Poison Javelin 0.6 s, Plague Javelin 4 s,
   Immolation Arrow 1.0 s, Valkyrie/Shadow Warrior/Shadow Master 6 s (V M1 "Cast Delay"): the delay starts at the action flag and cannot be reduced; only the
   action flag (the frame the spell fires) shortens with FCR. For rotation DPS check the action-flag table, not the frame count.
3. Frame lengths at 0% for the "standard cast" are class-specific; teleport and other no-damage skills use the same cast rate.
4. Practical targets (S, community): Sorceress 105 FCR for the big step to 8 frames, 200 for 7 (rarely reachable); Paladin Hammerdin 75 (10f) or 125
   (9f); Necromancer 75 (10f) or 125; Druid 163 is rarely reached (89 or 99 common); Assassin trap sins 65/102/174 (traps also use attack speed). (S)

## 3. Faster Hit Recovery (FHR): frames at each breakpoint (C, checked against V W1, V M1 Amazon rows)

Base formula `ceil(256*HitBase/floor(AnimSpeed*(50+E)/100))-1`. HitBase/AnimSpeed from M1: Ama 6/256; Ass 5/256; Bar 5/256; Dru 1h swing 7/248,
other 7/256; Werewolf 4/256; Werebear 5/184; Nec 7/256; Vampire 4/136; Pal spear/staff 5/192, other 5/256; Sor 8/256; mercs A1 6, A2 8, A3 9, A5 5.

| Class / form | Table (value%:frames) |
|---|---|
| Amazon / A1 merc | 0:11, 6:10, 13:9, 20:8, 32:7, 52:6, 86:5, 174:4 |
| Assassin / Barbarian / Pal (other weapons) / A5 merc | 0:9, 7:8, 15:7, 27:6, 48:5, 86:4, 200:3 |
| Druid human, 1h swinging (sword, mace, axe, wand, throwing axe, dagger*) | 0:14, 3:13, 7:12, 13:11, 19:10, 29:9, 42:8, 63:7, 99:6, 174:5 |
| Druid human, other weapons | 0:13, 5:12, 10:11, 16:10, 26:9, 39:8, 56:7, 86:6, 152:5 |
| Werewolf | 0:7, 9:6, 20:5, 42:4, 86:3, 280:2 |
| Werebear | 0:13, 5:12, 10:11, 16:10, 24:9, 37:8, 54:7, 86:6, 152:5 |
| Necromancer (human) | 0:13, 5:12, 10:11, 16:10, 26:9, 39:8, 56:7, 86:6, 152:5 |
| Necromancer vampire | 0:15, 2:14, 6:13, 10:12, 16:11, 24:10, 34:9, 48:8, 72:7, 117:6, 208:5 |
| Paladin spear/staff | 0:13, 3:12, 7:11, 13:10, 20:9, 32:8, 48:7, 75:6, 129:5, 280:4 |
| Sorceress / A2 merc | 0:15, 5:14, 9:13, 14:12, 20:11, 30:10, 42:9, 60:8, 86:7, 142:6, 280:5 |
| A3 merc | 0:17, 5:16, 8:15, 13:14, 18:13, 24:12, 32:11, 46:10, 63:9, 86:8, 133:7, 232:6 |

(*W1 lists daggers as 1h swung for the Druid.) Standard community targets (S): Barbarian/Assassin/Paladin 48 FHR (5f) or 86 (4f); Sorceress 86 FHR
(7f), Amazon 52 or 86 (S/U for the "needed" choice, exact need depends on build).
Stun rules (when a hit triggers recovery) are in `mechanics-combat.md` section 8 (V M1). Hit recovery cancels cast/attack/run; during it you
cannot act (V M1).

## 4. Faster Block Rate (FBR), C checked vs V W1 and V M1

`Block frames = ceil(256*BlockBase/floor(AnimSpeed*(50+E)/100))-1`; Holy Shield adds +50 to E (V M1).
Block base/speed (V M1): Ama 3/256 (1h swung weapons 3/88), Ass 3, Bar 4, Dru 6, Werewolf 5, Werebear 5/200, Nec 6, Pal 3, Sor 5.

| Class | Table (value%:frames) |
|---|---|
| Amazon, 1h swinging weapon | 0:17, 4:16, 6:15, 11:14, 15:13, 23:12, 29:11, 40:10, 56:9, 80:8, 120:7, 200:6 |
| Amazon, other weapons / Assassin / Paladin (no Holy Shield) | 0:5, 13:4, 32:3, 86:2 |
| Paladin with Holy Shield | 0:2, 86:1 |
| Barbarian | 0:7, 9:6, 20:5, 42:4, 86:3, 280:2 |
| Druid human / Necromancer | 0:11, 6:10, 13:9, 20:8, 32:7, 52:6, 86:5, 174:4 |
| Werewolf | 0:9, 7:8, 15:7, 27:6, 48:5, 86:4, 200:3 |
| Werebear | 0:12, 5:11, 10:10, 16:9, 27:8, 40:7, 65:6, 109:5, 223:4 |
| Sorceress | 0:9, 7:8, 15:7, 27:6, 48:5, 86:4, 200:3 |

Cross-check: almarsguides FBR tables exist for the same set (reported by fetch, not read in full; S). FBR is a shield-only affix (plus Guardian Angel and Pattern
per M1). Max block chance is 75% standing/walking and 25% running (see combat doc).

## 5. Increased Attack Speed (IAS)

### 5.1 Equation (V M4 Maxroll attack speed)
```
AnimDuration = (AnimLength * 256) / floor[ AnimSpeed * (AnimRate + SIAS + EIAS - WSM) / 100 ]  - 1   (round up the quotient)
  AnimRate = 100, AnimSpeed = 256 (class-specific exceptions below)
  EIAS = floor(120 * IAS_item / (120 + IAS_item))          (IAS on weapon + gear)
  WSM  = weapon speed modifier of the *primary* weapon (+20 slow .. -60 fast); two weapons averaged for dual-wield skills
  SIAS = skill IAS: Fanaticism, Burst of Speed, Werewolf, Frenzy (positive), Decrepify/chill/Holy Freeze (negative; slow capped at -50 on characters)
```
- Only the primary-hand weapon's IAS counts for single-weapon attacks (V M4). EIAS clamp -85..+75 (wereforms up to 150) (S W2).
- Throwing weapons used for a throw skill: flat 30 penalty to AnimRate (not for Barbarian) (V M4).
- **Sequence skills** (Fists of Fire, Claws of Thunder, Blades of Ice, Charge, Impale, Jab, Leap Attack) use `WSM+30` and no `-1` (V M4); Leap/Charge ignore WSM.
- **Rollback (multi-hit) skills**: Dragon Talon, Fend, Fury, Strafe, Zeal; each hit has a different length, no closed form; use a calculator such as D2Planner (V M4).
- Dual-wield (Assassin, Barbarian): secondary-hand IAS ignored; Double Swing `Vinc = 120 + (WSM1+WSM2)/2 + SIAS + EIAS`, `fpa = 256*17/(256*Vinc/100)/2`;
  Double Throw `Vinc = 70 + avgWSM + SIAS + EIAS`, `fpa = 256*12/(256*Vinc/100)/2`; Frenzy per-hand Vinc, with a bug when the first swing misses: the second swing uses the
  first weapon's values (V M4).
- Shapeshifted: Druid werewolf/werebear use `AnimSpeed = 256*FramesNeutral/Delay`, `Delay = 256*FramesChar/((100+WIAS-WSM)*CharSpeed/100)` with NU frames 13 (wolf)/12 (bear),
  unshifted attack frames 15 (V M4). Trang-Oul vampire form: length 14 speed 176 (D2R page; 1.14d numbers U). So off-weapon IAS matters less for druids (V M4).

### 5.2 Skill IAS values (S W2, D2R-derived calculator; formulas match 1.14d skill descriptions per our knowledge U)
`bonus = min(min + floor(factor*floor(110*lvl/(lvl+6))/100), max)`:
Fanaticism 10/30/40 (level 20: 35), Burst of Speed 15/45/60 (level 20: 52), Werewolf 10/70/80 (level 20: 68), Frenzy 0/50/50 (level 20: 42), Holy Freeze 25/35/50.
Maul (Werebear): `3*(floor(lvl/2)+3)`. Debuffs: Decrepify -50, chilled -50. Skill adjustments: Double Swing +20, Dragon Tail -40,
Jab/FoF/CoT/BoI/Dragon Claw/Frenzy/Double Throw -30.

### 5.3 WSM by base weapon (V A6, Arreat per-type weapon pages, normal-tier names; exceptional/elite counterparts have the same WSM row)

Negative = faster. (Class speed words on those pages are Fast/Very Fast/Normal/Slow; a few weapons are class-restricted, e.g. sabre fast classes.)

| Type | Weapon : WSM |
|---|---|
| Axes | Hand Axe 0, Axe +10, Double Axe +10, Military Pick -10, War Axe 0, Large Axe -10, Broad Axe 0, Battle Axe +10, Great Axe -10, Giant Axe +10 |
| Bows | Short Bow +5, Hunter's Bow -10, Long Bow 0, Composite Bow -10, Short Battle Bow 0, Long Battle Bow +10, Short War Bow 0, Long War Bow +10 |
| Crossbows | Light Crossbow -10, Crossbow 0, Heavy Crossbow +10, Repeating Crossbow -40 (exceptional/elite repeating crossbow: -60) |
| Daggers | Dagger -20, Dirk 0, Kris -20, Blade -10 |
| Javelins | Javelin -10, Pilum 0, Short Spear +10, Glaive +20, Throwing Spear -10 |
| Maces | Club -10, Spiked Club 0, Mace 0, Morning Star +10, Flail -10, War Hammer +20, Maul +10, Great Maul +20 |
| Polearms | Bardiche +10, Voulge 0, Scythe -10, Poleaxe +10, Halberd 0, War Scythe -10 |
| Swords | Short Sword 0, Scimitar -20, Sabre -10, Falchion +20, Crystal Sword 0, Broad Sword 0, Long Sword -10, War Sword 0, Two-Handed Sword 0, Claymore +10, Giant Sword 0, Bastard Sword +10, Flamberge -10, Great Sword +10 |
| Wands | Wand 0, Yew Wand +10, Bone Wand -20, Grim Wand 0 |
| Elite/exceptional notable | Phase Blade -30; Berserker Axe 0 (S W2 calculator notes); Phase Blade -30 matches V A6 elite swords list |

(Parsing caveat: V A6 pages list the WSM in a bracket per row; our row-to-name pairing came from the HTML; for spears/staves/scepters/throwing we did not
extract it and the numbers are U. Kris=-20 and Dirk=0 look plausible against known play.)
- Class speed words differ for some weapons (e.g. Short Sword "Fast" for Ama/Ass/Bar/Dru/Nec/Pal, "Normal" for Sor) in A6 text; that is class-dependent animation speed, not WSM.

### 5.4 Example from a calculator (S W2)
Amazon with a Phase Blade (WSM -30): 11 frames at 0 IAS, 10 at 6, 9 at 16, 8 at 30, 7 at 52, 6 at 89 (W2 example "an Amazon with a Phase Blade might see").
Rule: frames at 0 IAS equal the class/weapon-type animation length (FPD) minus the effect of WSM.
Base attack animations (FPD): Amazon unarmed 12 (V M4 worked example gives AnimDuration 11 with `-1`); Kick 13 (Assassin) /12 others; Smite 12; dodge/avoid 9; traps 8 at half speed (S W2).
A full per-class/weapon FPD table is in the game's PlrMode/CharStats data; we could not fetch a free page (diablowiki and purediablo are Cloudflare-blocked; see open questions). U for all
other base frame counts.

## 6. How the C tables were produced
Python re-implementation of the formulas of section 1: `E=floor(X*120/(X+120))`, `d=floor(spd*(K+E)/100)`, `frames=ceil(256*base/d - 1)`; scanned X=0..1000 and
recorded the X where frames change. Verified: Amazon bow FCR and Amazon FHR/FBR rows equal V M1; Barbarian/Sorc FCR (13..7), Assassin FCR (16..9), Amazon, Assassin, Paladin FHR,
Werewolf/Werebear/Vampire FHR/FBR and the Act 3 merc tables equal V W1 breakpoints. Bases for Druid/were-forms/Necro were fitted to W1 because M1's printed bases did not reproduce
its own tables (section 2 note 1).

## 7. Skill-specific / class-specific notes useful for a bot
- Hit recovery and cast frames interact: a stun (hit recovery) interrupts casting, so FHR/blocking-immunity matters for melee builds; casters usually want FCR then FHR (S).
- Teleport uses the cast rate; Sorceress blink-teleport cadence = cast frames (S).
- Blizzard/Meteor/Hydra casts are limited by cast delay not by frames (see section 2 note 2).
- Whirlwind uses truncation (floor) instead of ceil for frame count and the weapon type's action frame; Barbarian 2h swords treated as 1h swinging for WW (S W2).
- Zeal: rollback; breakpoints are best taken from a calculator (V M4).
- Run/walk (FRW): not fetched; U. Faster Run/Walk uses the same `120*X/(X+120)` family (U).

## Sources
- M1 Maxroll breakpoints & animations: https://maxroll.gg/d2/resources/breakpoints-animations (V, fetched with browser, "Last Updated Feb 10 2026", D2R).
- M4 Maxroll attack speed: https://maxroll.gg/d2/resources/attack-speed (V, same date).
- W1 Wowhead D2R breakpoints guide (S, D2R S13): https://www.wowhead.com/diablo-2/guide/breakpoints-by-class-fcr-ias
- W2 D2R IAS calculator reference (S, fan tool): https://d2r-ias-calculator.pages.dev/calculations
- A6 Arreat Summit weapon pages via Wayback, e.g. https://web.archive.org/web/2016id_/http://classic.battle.net/diablo2exp/items/normal/swords.shtml (V; same path for axes, bows, crossbows, daggers, javelins, maces, polearms, wands; elite/ and exceptional/).
- Asgard breakpoints guide https://asgard.pages.dev/guides/bp (S, not read in detail), Almar's Guides FBR (S).

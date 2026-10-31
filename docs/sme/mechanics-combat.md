# D2 SME KB: core combat mechanics (classic LoD 1.14d)

Author: SMEKB-A. Scope: Diablo II Lord of Destruction 1.14d (offline/open, classic). D2R and the 2026 Reign of the Warlock
(RotW) additions are flagged where a source mixes them in. Sibling docs: `mechanics-breakpoints.md`, `mechanics-progression.md`,
`mechanics-items-drops.md`.

Tags: **V** = fetched from a primary/reliable page (URL given), **S** = snippet or secondary/fan site, **U** = unverified or from memory.
All sources are listed in the Sources section at the end (A1.. etc.).

## 0. Reading guide and caveats

- Arreat Summit (A1) is Blizzard's own manual for 1.10-1.13. Where it conflicts with community datamining (Amazon Basin wiki
  = B1, Phrozen Keep derived), the datamined version is usually the better description of the engine, but the discrepancy is
  noted. (S)
- Amazon Basin wiki now also documents D2R and RotW ("Warlock" columns, "D2R only" items, "Ladder"); those columns are ignored here. (V B1)
- The Basin "Difficulty" page distinguishes "Expansion" from "Classic" (non-LoD) rules. This KB is Expansion/LoD only. (V B3)

## 1. Attack damage

### 1.1 Weapon damage and stat bonus (V A2, V B2)

```
Final min = (WeaponMin + flat min dmg from non-weapon items) * (1 + StatsBonus + EnhancedDmg%/100) * Modifier
Final max = (WeaponMax + flat max dmg ...)                  * (1 + StatsBonus + EnhancedDmg%/100) * Modifier
```
- `StatsBonus` (A2): hammers (War Hammer, Maul, Great Maul and upgrades) 1.10*Str/100; daggers, throwing weapons, claws
  0.75*Str/100 + 0.75*Dex/100; bows and crossbows Dex/100; Amazon spears/javelins 0.80*Str/100 + 0.50*Dex/100; everything else Str/100.
  Basin gives the same split per weapon class (V B2: Amazon bow Dex 100, Amazon javelin/spear Str 80 + Dex 50, claws/daggers/javelins
  /throwing 75+75, bow/crossbow Dex 100).
- Enhanced damage from weapon affixes applies to the weapon base directly; jewels in a weapon enhance the weapon base, while jewels
  in armor/helm/shield add to the same bucket as the stat bonus and skill % bonuses (V A2).
- `+x Damage` (e.g. The Redeemer) is flat min=max and is not shown on the character sheet (V A2).
- Modifier column (V A2): default 1; Multiple Shot / Strafe / Lightning Bolt 0.75; Critical Hit / Deadly Strike 2;
  Blade Fury 1h 0.75, 2h 0.375 (also Blade Sentinel); Blade Shield 0.25.
- No item needs more than 253 Str. Amazon bows never need more than 134 Str; spears/crossbows up to 152 (V A2).
- Ethereal weapons +50% base damage (V B2 list item "% Ethereal weapon bonus (+50%)"; the page lists the order of modifiers
  1 base, 2 ethereal, 3 ED% on weapon, 4 flat adds, 5 damage+, 6 %damage, 7 DS/CS/mastery, 8 unit crit, 9 elemental, 10 skill %,
  11 conversion, 12 source penalty). Order of operations is useful when reasoning about rounding but not decisive for play. (S)
- Wording quirk: weapons where min >= max get max = min+1 (ranged, bug) (V B2).

### 1.2 Damage types
Physical, Magic, Fire, Cold, Lightning, Poison. Poison is damage over time (length matters; "Poison Length Reduced" exists).
Cold also chills (slow, length). Bone spells and Berserk and curses are not "elemental" (V A5). Monsters that resist
Magic also resist Teeth/Bone Spear/Bone Spirit and Berserk (V A5).

### 1.3 Monsters in multiplayer / `/players X` (V A3)
- Monster HP = base * (n+1)/2 where n = player count, and XP likewise = base * (n+1)/2 (A3). Basin lists XP 100%, 150% ... 450%
  at 1..8 players (V B4).
- `/players X` works in single player, open Bnet and TCP/IP; it raises monster life, experience and drops (V A3). Item count only
  rises at odd counts (see items doc).

## 2. Attack Rating, Defense, chance to hit

### 2.1 Formulas
- Per Dexterity point: +5 Attack Rating, +1 Defense per 4 Dex (V B5 text "every additional point of Dexterity also results in +5 AR ...
  every four points results in +1 Defense"). Naked Defense = floor(Dex/4) (V A2). Base AR = (Dex-7)*5 + class constant
  with constants Ama 5, Ass 15, Bar 20, Dru 5, Nec -10, Pal 20, Sor -15 (V B6: level-1 AR Ama 95, Ass 80, Bar 85, Dru 70, Nec 80,
  Pal 85, Sor 75 with their starting Dex).
- **Chance to hit (melee/missile hit check)**: `CTH% = 100 * AR/(AR+DR) * 2*alvl/(alvl+dlvl)`, clamped to 5%..95% (V A2).
  A2 additionally lists a "class bonus to level of attacker" (Bar/Pal +20, Ass +15, Ama/Dru +5, Nec -10, Sor -15). (V that Arreat
  prints it; S that it is a mis-description. B6 says the displayed chance to hit "applies the class constant a second time",
  i.e. the constants above are the AR class constants, not level bonuses. Treat the Arreat level-bonus reading as doubtful.)
- Monsters use the same formula against the player's Defense. In Nightmare/Hell monster level >> player level hurts melee both
  ways (V A4 "better chance to hit you, harder time hitting them").
- Rule of thumb from Arreat (V A2): melee needs roughly 50-75% AR chance to hit reliably.
- Defense only reduces chance to be hit, not damage (V A2).
- Skills that never use AR: magical attacks and non-combat skills (V A2). Some attack skills ignore defense or add AR
  (e.g. Zeal/Sacrifice/Smite details): U.

### 2.2 Worked example (computed, formula V A2)
Level 30 Paladin AR 800 vs monster Defense 400, monster level 36: 100*(800/1200)*2*30/66 = 60.6%. Raising AR to 1600:
100*(1600/2000)*0.909 = 72.7%. Gains flatten as AR >> DR. (computed)

## 3. Blocking

- Block is rolled after a hit check succeeds and before dodge/avoid/evade (V B7). Prevents damage and modifiers.
- Blockable: melee, arrows/bolts, javelins, throwing weapons/knives, physical missiles, Bone Fetish and Suicide Minion death
  explosions, Fire Enchanted explosion. Not blockable: Fire Arrow/Cold Arrow above level 49, Lightning Bolt, Molten Boulder
  explosion, Tornado radial damage, Armageddon radial, Corpse Explosion, Paladin Smite, charging Claw Vipers, Reanimated Horde,
  Nihlathak corpse explosion (V B7). Spell projectiles (fire bolt etc.) are not "physical missiles".
- **Expansion formula** (V A2, V B7): `Block% = min( (ShieldBlock + BonusBlock) * (Dex-15) / (clvl*2), 75 )`.
  Classic (non-LoD) has no Dex scaling (B7). Required Dex for the 75% cap: 15 + 150*clvl/(Shield+Bonus) (V B7).
- Running reduces block to 1/3 after the cap (max 25%); walking, standing, casting and Whirlwind keep the full value (V A2, V B7).
- Class bonus to shield block (already baked into the shield table on B7): Amazon/Assassin/Barbarian +25, Druid/Necromancer/Sorceress +20,
  Paladin +30. Example base-shield rows on B7 (Dru/Nec/Sor | Ama/Ass/Bar | Pal): Buckler 20|25|30, Small 25|30|35, Large 32|37|42,
  Kite 28|33|38, Spiked 30|35|40, Bone 40|45|50, Tower 44|49|54, Gothic 36|41|46 (V B7).
- Amazon Avoid/Dodge/Evade and Assassin Weapon Block are only checked if the shield block roll fails or no shield (V A2).
- Faster Block Rate: breakpoint tables in `mechanics-breakpoints.md`. Block animation makes you immune to further physical hits for its duration
  (S Asgard guide A8; U for exact mechanics).

## 4. Critical-type effects

### 4.1 Deadly Strike / Critical Strike / Weapon Mastery crit (V B8)
- Deadly Strike (DS) doubles physical damage on hit; it can also double converted damage (Magic Arrow, Fire Arrow, Cold Arrow,
  Lightning Bolt, Fists of Fire, Berserk incl. synergies) because conversion happens after the double. Does not apply to Impale, Dragon
  Talon/Tail/Flight, Sacrifice, Smite.
- DS from multiple items sums (cap 100%). DS, Amazon Critical Strike and weapon/claw mastery crit are mutually exclusive: combined
  chance = 100 - (100-DS)*(100-CS)/100 (two sources); with three: 100 - (100-DS)(100-CS)(100-MCS)/10000.
  Example: DS 80% -> 100% raises average physical damage from 180% to 200% (+11%). (V B8)
- Barbarian merc, Valkyrie and Iron Golem have a separate 5% chance of a crit that doubles *all* damage types (can quadruple physical).
- Two-weapon: only the attacking weapon's DS counts (V B8).

### 4.2 Crushing Blow (V B9)
Applied before the hit's damage. Removes a fraction of the target's *current* life (monster base life without the per-player scaling):

| Target | Melee | Ranged |
|---|---|---|
| Normal/minion/champion/unique monsters, pets | 1/4 | 1/8 |
| Super uniques and bosses | 1/8 | 1/16 |
| Players and mercenaries | 1/10 | 1/20 |

- Chance from multiple items sums, cap 100%. Not doubled by DS. Not applied by Blade Sentinel/Shield or extra Multiple Shot arrows.
- In multiplayer the damage is divided by 0.5 + 0.5*players, so it cuts base (unscaled) life by the fraction (V B9).
- Reduced by monster Damage Resist (DR>99% = immune; negative DR does not amplify). Physical-immune undead are immune
  to CB. Not reduced by Bone Armor/Energy Shield/player DR cap rules (V B9).
- Practical (U): against bosses each CB proc removes 1/8 (melee) of current life, so CB is strong against high-HP bosses.

### 4.3 Open Wounds (V B10)
- Target bleeds for 8 s (200 frames); stops natural regeneration; works even if life drain cannot apply. Always applied to players;
  on monsters only if their life changes.
- Life drain per frame in 1/256 life by attacker level L: L1-15 `9L+31`; 16-30 `18L-104`; 31-45 `27L-374`; 46-60 `36L-779`;
  >60 `45L-1319` (V B10). Per second = value*25/256. Examples: L75 -> (45*75-1319)=2056/256 per frame*25 = ~201 life/s on normals,
  halved vs "special units" (champion, unique, super unique, boss, evil); player target x1/4 melee, x1/8 ranged. (computed)
- Chance sums across items. Not applied by Blade Sentinel/Shield or extra Multiple Shot arrows.

## 5. Resistances, absorb, damage reduction

### 5.1 Resistances (V A5, V B3, V Maxroll M3)
- Four displayed resists: Fire, Cold, Lightning, Poison. Hidden: Magic (base 0 in all difficulties; no -40/-100 penalty;
  only Crafted Safety Shield gives +5..10%, V A5) and Physical (shown as "Damage Taken Reduced by x%", see below).
- **Difficulty penalty on player and mercenary resists (Expansion): Normal 0, Nightmare -40, Hell -100.** Classic (non-LoD) is -20 / -50
  (V B3). Floor -100%. Same penalty applies to Poison Length reduction. Penalty is on the displayed resist, so reaching 75% in
  Hell needs 175 raw resist from gear/skills/quests (computed; consistent with V M3 "in Hell without any additional resistances characters
  take twice the damage").
- Default max resist 75%; items/skills can raise caps to 95% (V A5). Resist Fire/Cold/Lightning (Paladin) gives +1% max resist per two skill points;
  Guardian Angel +15 max resists on the four screens (V M3; M3 is D2R-era text, so U whether that item exists in 1.14d; the Paladin Resist skills do).
- Over-capping matters: Conviction and Lower Resist subtract from your effective resist, which is reduced from the cap (V M3).
- Resist All / Scroll of Resistance (Anya, Act 5 Prison of Ice quest): +10 to the four resists per difficulty,
  +30 over a full game (V M3). Bug: after death the bonus disappears until a new game is joined, though the sheet still shows it
  (V A5, V M3).
- Cold resist also scales chill/freeze length (V M3); Amulets/items "Half Freeze Duration" apply once only, "Cannot Be Frozen" sets 0 (V A5).
- Negative-resist items exist (e.g. Andariel's Visage -30 fire) (V M3 -- D2R page, 1.14 item list is the same for classics, S).

### 5.2 Physical damage reduction (V A5, V M3)
- "Damage Taken Reduced by x%" (DR%, physical resist): starts at 0, not penalised by difficulty, capped at 50% (cannot be raised).
  Amplify Damage lowers it by 100, Decrepify by 50, floor -100 (V A5, V M3).
- Flat "Damage Reduced by x" is separate and applies earlier (see order).

### 5.3 Absorb (V A5, V M3)
- Absorb heals first, then reduces damage; "% absorb" capped at 40%; flat "+x absorb" has no cap. Cannot absorb physical (no % physical absorb);
  magic only has flat absorb.
- Life loss = Dmg*(1-res)*(1-min(absorb%,40%)) - flatAbsorb - heal(%part) - heal(flat). Example M3: 1000 lightning, 50% res, 50% absorb,
  +20 flat -> 60 life lost. Absorb heals do not overheal; if resisted damage exceeds life you die before absorb reduces it (V A5 note).
- Arreat damage-reduction order (V A5): Energy Shield -> Bone Armor/Cyclone Armor -> DR and MDR (flat) -> Resistances -> % Absorb -> flat Absorb.

### 5.4 Monster resists by difficulty (S/U)
- Monster elemental resist tables live in `monstats.txt` per difficulty (Normal/NM/Hell columns). Typical shape (U, community knowledge):
  almost no immunities in Normal, immunities appear from Nightmare (Arreat: "more and more monsters with immunities", V A4), and in
  Hell a large fraction of Act 3-5 monsters are immune to one element; ~100% resist (immune) is the usual definition, monsters can have
  resist above 100 (still immune). Unique "Spectral Hit", "Fire/Cold/Lightning/Mana Burn Enchanted" bosses add elemental damage (V A5).
- Conviction/Lower Resist/Cold-Poison pierce are the standard counters; "pierce" skill/item stats cut effective resist for the
  attack's element and can break non-immune resists only; immunity (>=100%) is broken by pierce only partially (U, see build docs).
- Melee/phys-immune monsters (e.g. many Act 5 Hell monsters) force a damage-type switch (U).

## 6. Difficulty combat modifiers (V A4, V B3)

| Effect | Normal | Nightmare | Hell |
|---|---|---|---|
| Player/merc resist penalty (Expansion) | 0 | -40 | -100 |
| Freeze/chill/cold length | x1 | x1/2 | x1/4 (B3) |
| Life steal / mana steal received by player and merc | x1 | x1/2 | x1/3 (B3) |
| XP lost on death (of current level span) | 0 | 5% | 10% |
| Unique/champion damage bonus (A4) | +90% | +75% | +66% (V A4 as printed; the numbers read as damage bonus relative to the type; U whether 1.14d keeps them) |
| Hireling/summon damage vs act bosses | 50% | 35% | 25% (A4 says 50/35 and B3 says mercenary vs "Evils" 50/35/25) |
| AI-changing curses (Dim Vision, Terror, Confuse, Attract) duration | x1 | x1/2 | x1/4 (B3; A4 says 1/2 in NM) |
| Pet skill level bonus | +0 | +3 | +7 (B3) |
| Monster skill level bonus | - | +3 (A4) | not stated in A4 (U) |
| Static Field cannot reduce below | - | 33% | 50% (B3, Expansion) |
| Nihlathak corpse explosion damage | 50% | 35% | not stated (A4 gives 50/35 only) |

Other (V A4): NM/Hell monsters have higher level, HP, defense, XP, damage, AR and resists; unique monsters gain an extra
bonus ability; monster populations include guests from other acts; many areas get larger maps; Act 5 Barbarian soldiers have reduced HP.
B3 also has a row "Maximum sockets" with values 1 / 3 / 4 / 6 for already-socketed drops; its columns are ambiguous so treat as U.

## 7. Life, mana, stamina

### 7.1 Per class (V B11 life, V B12 mana, V B13 stamina, V B14 vit/dex/energy)

| | Ama | Ass | Bar | Dru | Nec | Pal | Sor |
|---|---|---|---|---|---|---|---|
| Start Str | 20 | 20 | 30 | 15 | 15 | 25 | 10 | (U)
| Start Dex | 25 | 20 | 20 | 20 | 25 | 20 | 25 | (V)
| Start Vit | 20 | 20 | 25 | 25 | 15 | 25 | 10 | (V)
| Start Ene | 15 | 25 | 10 | 20 | 25 | 15 | 35 | (V)
| Start Life | 50 | 50 | 55 | 55 | 45 | 55 | 40 | (V)
| Life/level | 2 | 2 | 2 | 1.5 | 1.5 | 2 | 1 |
| Life/Vit pt | 3 | 3 | 4 | 2 | 2 | 3 | 2 |
| Start Mana | 15 | 25 | 10 | 20 | 25 | 15 | 35 |
| Mana/level | 1.5 | 1.5 | 1 | 2 | 2 | 1.5 | 2 |
| Mana/Ene pt | 1.5 | 1.75 | 1 | 2 | 2 | 1.5 | 2 |
| Start Stamina | 84 | 95 | 92 | 84 | 79 | 89 | 74 |
| Stamina/level | 1 | 1.25 | 1 | 1 | 1 | 1 | 1 |
| Stamina/Vit pt | 1 | 1.25 | 1 | 1 | 1 | 1 | 1 |

- 5 stat points and 1 skill point per level, plus 5 extra stat points once from a quest reward (V A2: "5 additional points as a reward for completing one of the Quests"; which quest: U, the Lam Esen's Tome in Act 3).
- Displayed values are floored; internal unit is 1/256 (V B11). Max life from levels only, e.g. level 99 Barb: 55+2*98=251 + vit.
- Life = start + lvlGain*(clvl-1) + vitPoints*perPoint + item/charm life; percentage life buffs (e.g. Battle Orders) then multiply. Example (C): level 85 Barbarian with 200 vitality points above start: 55 + 2*84 + 200*4 = 1,023 base before items. (Vitality from items also counts per point at the class rate: U.)
- Level-up restores life, mana, stamina fully (V B15). A well in the field heals 50% for player + pets; healer in town full;
  Full Rejuvenation 100%, Rejuvenation 35% (V B11).

### 7.2 Regeneration
- Mana: empty to full in 120 s without items/Warmth. Per second = `25 * [[256*maxMana/(25*120)] * (100+%regen)/100] / 256` (floors in the
  brackets). 900 mana + Warmth 258% = 26.56/s (V A2). Approx maxMana*R/120.
- Replenish Life: `25*N/256` life/s (V A2); approx N/10. Open Wounds stops regen (V B10).
- Stamina: standing recovers max/256 per tick, walking half. Drain while running = `25*(40*(1+floor(armorSpeed/10))*(1 - slowerDrain%)/256)` per second
  (3.91/s in light armor) (V A2). Heavy armor/shield: medium/heavy armors drain faster and reduce walk/run speed; heavy shields -10% speed (V A2).
  Stamina potion: 30 s unlimited stamina, stacks (V A2, V D1).

### 7.3 Potions (V D1 Potions page, V A2)
Healing amounts differ by class: Amazon/Paladin/Assassin = base; Barbarian = 4/3 of base; Necromancer/Sorceress/Druid = 2/3 (V D1).
Life gradually fills over a class-independent time:

| Potion | Ama/Pal/Ass | Bar | Nec/Sor/Dru | Fill time |
|---|---|---|---|---|
| Minor Healing | 45 | 60 | 30 | 7.68 s |
| Light Healing | 90 | 120 | 60 | 6.4 s |
| Healing | 150 | 200 | 100 | 6.84 s |
| Greater Healing | 270 | 360 | 180 | 7.68 s |
| Super Healing | 480 | 640 | 320 | 10.24 s |

Mana potions (all fill in 5.12 s): Minor 30/20/40, Light 60/40/80, Normal 120/80/160, Greater 225/150/300, Super 375/250/500
(Ama/Pal/Ass | Bar | Nec/Sor/Dru) (V D1).
- Potions heal over time and overlapping potions average their remaining fill: a Greater plus Minor drunk together is slower than
  Greater alone (V D1). Rejuvenation (35% of both, instant) and Full Rejuvenation (100% instant) are not subject to doubling (V D1).
- **Double fill**: roll A in [0,Vit-1], B in [0,99]; if A>B the potion gives double. Chance: Vit even <=200: (Vit-2)/4 %;
  odd: (Vit-1)^2/(4Vit); above 200: 100*(Vit-101)/Vit. Mana uses Energy (V A2, V D1). 100 Vit = 24.5%, 200 = 49.5%, 400 = 74.8%.
- Antidote: cures poison + 50% poison resist and +10 max resist for 30 s (stacks duration). Thawing potion: same for cold; not vs Holy Freeze (V D1).
- Healing is the main tool until life leech/Replenish; belts hold the potions (slots below).

### 7.4 Belts (V A9 Arreat belts)
Belt potion rows: Sash, Light Belt = 8 slots (2 rows); Belt, Heavy Belt = 12 (3 rows); Plated Belt = 16 (4 rows); **all exceptional
(Demonhide Sash, Sharkskin, Mesh, Battle, War Belt) and elite belts (Spiderweb Sash, Vampirefang, Mithril Coil, Troll Belt, Colossus
Girdle) = 16 slots** (V A9). Only the bottom row is visible; items above drop down as you drink (V D1). Plated Belt needs 60 Str (V A9).
- Practical (U): rejuvenations are the usual top-row emergency potions; drink instant rejuvs before healing potions when life drops fast, because normal potions fill over several seconds.

## 8. Hit recovery (stun) trigger rules (V M2, Maxroll breakpoints page)
When a unit takes a hit it goes into hit recovery only if: damage did not kill it; it is not frozen; damage was not poison-only
and was at least 1; then `trigger = maxLife / divisor` where divisor is set by the missile's hit class (2,6,10,11 -> 8; 4,8 -> 32; 5 -> 64;
default 16). Damage < trigger: no stun; < 2*trigger: 50% stun; < 4*trigger: 75%; >= 4*trigger: certain. Some monsters are immune to
stun. Stun-lock is possible because stun can re-trigger without cooldown (V M2). FHR breakpoints are in the breakpoints doc.

## 9. Death penalty (V A2, V B3, V D2)
- Realm/multiplayer: lose gold = clvl% of carried+stash gold, capped 20% (V A2, V D2); the rest of carried gold drops on the corpse. Single player:
  stash is never touched and 500 gold per clvl is exempt (V A2, V D2).
- Nightmare/Hell: lose 5%/10% of the exp needed for the current level->next span, never a level; 75% of that is returned when you
  recover your corpse without leaving the game (V A4, V B15). Normal difficulty: no XP loss.
- Corpse holds equipped items; up to 16 corpses; in multiplayer the most valuable corpse persists across games (V A2).
- Equipment-order warning: Str/Dex items must be placed first (V A2).

## 10. 1.14d vs D2R notes
- All formulas above (damage, CTH, block, CB/DS/OW, resist order, XP, potions) are unchanged in 1.14d vs D2R as far as the cited
  community pages show; the Maxroll/Basin pages are written for D2R but flag changes explicitly. (S)
- D2R-only: ladder items and runewords, higher stash, new uniques (Basin tags "Yes (D2R only)"). RotW adds a Warlock class and Colossal jewels (B1 tag RotW). Out of scope here.
- Life/mana numeric globe display persists after click since 1.13 (V B11).
- Patch 1.11+ changed party XP to 87/256 per extra member (V B15).

## Sources
- A1 Arreat Summit, via Wayback (classic.battle.net, 2016 captures). Experience: https://web.archive.org/web/2016id_/http://classic.battle.net/diablo2exp/basics/experience.shtml ; Resistances .../basics/resistances.shtml ; Difficulty .../basics/difficulty.shtml ; Vendors .../basics/vendors.shtml ; Gambling .../basics/gambling.shtml ; Characters .../basics/characters.shtml (V).
- A2 = Arreat characters page (formulas for damage, AR, CTH, block, potions, mana, stamina).
- A3 = Arreat experience page (party, player counts). A4 = Arreat difficulty page. A5 = Arreat resistances page.
- A8 Asgard PvP breakpoints guide https://asgard.pages.dev/guides/bp (S, D2R PvP-focused).
- A9 Arreat belts: https://web.archive.org/web/2016id_/http://classic.battle.net/diablo2exp/items/normal/belts.shtml (and exceptional/, elite/) (V).
- B1..B15 Amazon Basin wiki https://www.theamazonbasin.com/wiki/index.php?title=PAGE : Attack_Rating (B6), Dexterity (B5), Block (B7), Deadly_Strike (B8),
  Crushing_Blow (B9), Open_Wounds (B10), Life (B11), Mana (B12), Stamina (B13), Vitality/Energy (B14), Experience (B15), Difficulty (B3), Damage (B2), Experience (B4 same page). (V, page mixes D2R/RotW).
- M2 Maxroll breakpoints & animations https://maxroll.gg/d2/resources/breakpoints-animations (V text via browser fetch, Feb 2026, D2R). M3 Maxroll damage reductions https://maxroll.gg/d2/resources/damage-reductions (V).
- D1 Diablo Wiki Potions https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/Potions ; D2 Diablo Wiki Gold https://web.archive.org/web/2023id_/https://diablo2.diablowiki.net/Gold (V, Wayback).

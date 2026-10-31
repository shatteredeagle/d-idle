# Assassin plan (D2 LoD 1.14d, offline softcore, unattended)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** unverified (memory or judgement). Guide pages are Maxroll D2R Season 14; items/effects that exist only in D2R are marked D2R. Prefer values from the game's .txt tables at setup.

Sources: Maxroll Assassin leveling https://maxroll.gg/d2/guides/assassin-leveling (V), Lightning Sentry Assassin https://maxroll.gg/d2/guides/lightning-sentry-assassin (V), build list https://maxroll.gg/d2/build-guides (V), Maxroll immunities https://maxroll.gg/d2/resources/immunities (V, D2R), Kolbot `Attacks/Assassin.js` https://github.com/blizzhackers/kolbot (V).

## 1. Build choice

| Build | Safety | Mana | Gear need | Immunities | Automation | Verdict |
|---|---|---|---|---|---|---|
| Fire Blast + Wake of Fire (leveling, V) | Medium (traps at range, Wake cast near) | OK | Low | Fire immune | Ground/target cast | **Normal to early Nightmare** |
| Lightning Sentry + Death Sentry + Charged Bolt Sentry (Trapsin) | Medium-high (V "lay down traps from a distance, they attack even off-screen") | Low (teleport 1 mana at high level, V) | High end gear for Hell | Lightning immune (V), merc/Death Sentry/Fire Blast cover | Placement logic, 5 trap cap | **Primary** |
| Martial arts (Phoenix Strike, Dragon Talon/Tail) | Low-medium (melee) | OK | High | Physical/elemental mix | Charge-up combos: needs two-step attack timing | rejected (input complexity) |
| Whirlwind / Blade / Rift Assassin | low | | | | | rejected (tier D in the list, S) |

Maxroll's plan: Fire Blast / Wake of Fire until clvl 38, then respec into traps (V). The respec point is a confirmed inconsistency in the page (36 vs 38, V).

## 2. Level ranges, areas and runs

From the Maxroll page (V): Act 1 Normal to clvl 13 (Den of Evil, then Countess for runes), Acts 2-5 Normal to 26 (Radament, Izual, Baal; Cow Level for a sword base), Acts 1-3 Nightmare to 38 (Countess in Nightmare, Act 5 quests), respec at the start of Act 4 Nightmare, Act 4 Nightmare through Act 5 Hell to 75. Hell XP spots: use `docs/plan/data/progression-hell.json`.

Quest/run additions: Act 5 Quest 1 socket reward for Spirit; skip Baal wave 5 (V: restock from Malah and go on). The Act 2 mercenary is "needed for Grandvizier of Chaos and Colenzo" (V) because those are Lightning immune.

## 3. Skill order and stats

- Levels 2-5 Fire Blast, Claw Mastery at Den of Evil; 6-11 Burst of Speed, Fire Blast at 8; 12-31 Wake of Fire (Fire Blast on bosses); 32-38 Fire Blast (V).
- Respec at 36-38 (V): Cloak of Shadows 1, Mind Blast 1, Burst of Speed 3, Lightning Sentry 15, Death Sentry 9, Charged Bolt Sentry 10. Max order Lightning Sentry, Charged Bolt Sentry, Shock Web, Fire Blast.
- Stats (V): 2-3 Energy, 4-10 Vitality, 11 Strength, then Vitality; 22-25 add 18 Strength; 48 Strength if a Broad Sword is used for Spirit. Late: Strength for gear, rest Vitality; Dexterity only for 75% block (hardcore).

## 4. Gear and runewords

- Stealth (Tal Eth Tir Ral in V text, which lists four runes; the standard recipe is Tal+Eth, treat as U) by clvl 17; Leaf short staff (Tir Ral) at 19; Spirit sword after Nightmare Countess; Lore helm (Ort Sol); Ancients' Pledge kite shield (V).
- Shopping (V): belt from Farah, Battle Staff of Teleportation from Ormus (clvl 20-23), Wand of Lower Resistance from Akara/Drognan at clvl 30-35, Nightmare belt/boots/gloves for lightning and fire resists.
- Endgame trap gear (V): +Lightning Sentry weapons, -x% enemy lightning resistance, Griffon's Eye, Magefist, Arachnid Mesh, Harlequin Crest. Crack of the Heavens and the Patch 2.6 trap lightning-resist reduction are D2R-only or patch-2.x changes (V Maxroll, which says Patch 2.5/2.6); do not plan on them for 1.14d. Charged items such as the Staff of Teleportation recharge when the town blacksmith repairs them (V Arreat Summit https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/items/magic/chargedsuffixes.shtml); the number of charges and Teleport level scale with the item's ilvl (V same page), exact Ormus staff values are not found (U).

## 5. Mercenary

Act 2 Desert Mercenary with Holy Freeze and Infinity polearm for Conviction (V). Hire after entering Act 4 and equip a Pike from Halbu (V). In Hell the merc is what kills trap-immune packs (V), so keep him alive and near enemies; teleport takes him along but traps stay behind (V).

## 6. Dangers

- Normal: weak early Act boss farming (V). Wake of Fire "struggles in narrow corridors" (Maggot Lair, Arcane Sanctuary, V).
- Nightmare/Hell: lightning-immune packs and bosses (Grandvizier of Chaos always immune; Madawc always immune; seal bosses in the Chaos Sanctuary, V). Fix: Wand of Lower Resistance or Infinity merc; Death Sentry detonating corpses (half fire, half physical, V); kill with merc.
- Madawc and elite affixes to avoid: Magic Resistant, Lightning Enchanted, Spectral Hit (V).
- Class-specific death causes (U): casting traps with a 5 cap while monsters melee her; low life pool; Burst of Speed and Fade are mutually exclusive (V), so resist-capping is a swap decision.

## 7. Mana, potions, charms

Low mana need once Teleport cost falls to 1 (V); Insight merc weapon helps (V). Belt per `potions-belt.json`. Charms: Small Charms of Vita, Grand charm with +1 traps (V), resist charms. Cannot-be-frozen (Hearth) keeps trap laying fast (V).

## 8. What the bot needs (vs the Sorceress)

| Topic | Sorceress (current) | Assassin |
|---|---|---|
| Cast mode | right-click `CastAt` unit/ground | `CastAt(x,y)` ground for traps; traps are placed near the target (V Kolbot `placeTraps` places around `unit.x/y`, `trapRange` 20) |
| Trap bookkeeping | none | read own trap count (5 max, V Kolbot via minion count of AssassinTrap), re-place when the pack moves > 15 (V Kolbot) |
| Position | keep distance 10 subtiles | stay at about 20 subtiles for placement, then retreat; no melee |
| Buffs | Frozen Armor | Burst of Speed (speed) vs Fade (resist) toggle, Cloak of Shadows (needs cooldown tracking, V Kolbot gates by `me.skillDelay`), Mind Blast, Shadow Master (shares the 5 trap cap: avoid, V) |
| Immunity | cold, lightning, fire | lightning is main; secondary Fire Blast and merc. Need a "skip pack" decision |
| Weapon swap | none | Staff of Teleportation charges, swap weapons for teleport (also Kolbot `TeleSwitch`, V) |
| Existing code to change | `PickAttack`, `Layout` | trap placement planner (new), ground `CastAt` with offsets, own-summon count, charge item usage |

Delta over the Sorceress: medium-high (trap planner). Martial arts would add charge/finisher timing and would be higher.

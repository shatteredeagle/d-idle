# Amazon plan (D2 LoD 1.14d, offline softcore, unattended)

Tags: **(V)** fetched from a primary/community-reliable page this session, **(S)** search snippet or secondary, **(U)** unverified, from memory or design judgement. Most guide pages are Maxroll for D2R Season 14; mechanics below are 1.14d unless noted. D2R-only items (Sunder Charms, Flame Rift/Bone Break style charms, Patch 2.4+ monster changes, mercenary gold cap rules) are marked D2R. Read values from the game's own .txt tables at setup, not from here (same rule as `docs/plan/leveling.md`).

Sources: Maxroll Lightning Fury Amazon https://maxroll.gg/d2/guides/lightning-fury-amazon-guide (V), Maxroll Amazon overview https://maxroll.gg/d2/resources/amazon-overview (V), Maxroll build list https://maxroll.gg/d2/build-guides (V), Maxroll immunities https://maxroll.gg/d2/resources/immunities (V, D2R), Kolbot `Attacks/Amazon.js` https://github.com/blizzhackers/kolbot (V), Icy Veins / D2DB / Warcraft Tavern snippets for respec levels (S).

## 1. Build choice

| Build | Safety | Mana | Gear need | Immunities | Automation | Verdict |
|---|---|---|---|---|---|---|
| Bow/javelin: Fire Arrow, Poison Javelin, Exploding Arrow (to clvl 38), then Lightning Fury + Charged Strike (V) | Medium-high (ranged, Valkyrie, Decoy) | Poor for Fury (V: "high mana costs") | High at the end (Pierce, +skills, Insight merc) | Lightning immune is "most Hell areas" (V); Jab as backup | Fury is an area cast at a target; Charged Strike is melee-range | **Primary** |
| Exploding Arrow / Immolation Arrow bow | Medium (ranged) | Poor | Medium | Fire immune | Targeted arrow, simple | **Early path** (Maxroll leveling uses Exploding Arrow, V) |
| Multiple Shot / Strafe (Guided Arrow) | Medium | OK | Very high (V: "not recommended to start") | Physical immune | Strafe is hold-to-cast | rejected |
| Freezing Arrow | Medium | Poor | High | Cold immune | Targeted | rejected |
| Fend (Jab) | Low (melee) | OK | Medium | Physical | Melee positioning | rejected |

Choice: javelin line. Poison Javelin is a single point-and-click targeted skill that needs no aiming mechanics (U: skill id "Poison Javelin", mana cost low, poison does damage over time and hits through the target's physical immunity but not poison immunity). Lightning Fury in late game needs Pierce and +skill gear; the Maxroll build assumes character level 75 before switching (V). Amazon is "among the slowest characters" and has no native mobility (V); teleport needs Enigma or a charged Staff of Teleportation, and her low cast rate makes it painful (V).

## 2. Level ranges, areas and runs

Source for this section: Maxroll Amazon leveling guide https://maxroll.gg/d2/guides/amazon-leveling-guide (V, fetched; written for D2R Season 14, the areas and runewords are the same in 1.14d except D2R-only balance).

| clvl | Where | Notes |
|---|---|---|
| 1-13 | Act 1 Normal: Den of Evil, Countess in Black Marsh (runes), Andariel | Fire Arrow, Poison Javelin at 6, Multiple Shot at 11, Exploding Arrow 12-13 (V). Hunter's Bow with Rubies, Stealth in a Studded Leather, Gemmed/Lore caps (V) |
| 14-26 | Acts 2-5 Normal: Radament, hire the Act 2 merc after entering Act 4, Cow Level for gold/XP, Eldritch at 25 | Exploding Arrow; Staff of Teleportation from Ormus at clvl 20-23 (V) |
| 27-38 | Acts 1-3 Nightmare: Countess for Lore runes (Shael Eth Ort Sol); back to Act 5 Normal at about clvl 29 for Javelins of Swiftness | Exploding Arrow to 31, Fire Arrow 32-38; Wand of Lower Resistance bought at 30-35 (V) |
| 38 | **Respec** | Pierce 5, Dodge/Avoid/Evade 1 each, Lightning Fury 9, Charged Strike 19 (V) |
| 39-75 | Act 4 Nightmare to Act 5 Hell | Fire immunity in Act 4 Nightmare is why the respec happens there (V). Hell areas: `progression-hell.json` |

Stats (V): 2-11 Vitality; 12 three Dexterity and two Strength; 13 Strength; 14-16 Energy; 17-38 Vitality; after respec Strength for gear, Dexterity for 50% block, rest Vitality. The earlier tertiary respec figures (31-35) are superseded.

Runs worth adding for Amazon over the Sorceress plan: none fundamentally; Cow Level is a good Plague/Lightning Fury farm (V overview) but needs the Wirt's Leg quest (U).

## 3. Skill order and stats

- Early (V Maxroll amazon-leveling-guide): Fire Arrow 2-5, Poison Javelin 6, Multiple Shot 11, Exploding Arrow 12-13 and onward. Kolbot uses Inner Sight on act bosses because they are immune to Slow Missiles (V Amazon.js). Valkyrie needs clvl 30 (U).
- Late (V Maxroll Fury): max Lightning Fury, Charged Strike, Lightning Strike, Power Strike, Lightning Bolt. 1 point each: Jab, Pierce, Dodge, Avoid, Evade, Slow Missiles, Decoy, Valkyrie.
- Stats (V): enough Strength/Dexterity for gear, rest Vitality. Dexterity for 75% block only with a shield build (V).
- Same structural plan as the Sorceress: put data in `data/build-*.json` form; we need one primary and one fallback spec.

## 4. Gear and runewords

- Early runewords the generic Maxroll pages use for every class (V Assassin/Druid/Necro pages): Stealth (Tal+Eth) at clvl 17 in a 2-socket body armor, Ancients' Pledge (Ral Orth Tal) in a shield, Spirit (Tal Thul Ort Amn) in a 4-socket sword, Lore (Ort Sol) in a helm. Runewords do not work in magic bases (V).
- Amazon-specific: Amazon bows, javelins and spears spawn with +1 to +3 skills (V overview). Weapon-swap Harmony Hunter's Bow or Staff of Teleportation (V Starter build). Wand of Lower Resistance for immunities (V). Razortail belt, Titan's Revenge, Thunderstroke, Crack of the Heavens (D2R-only effect: sets Lightning immune to 95%) in the endgame build (V). In 1.14d Crack of the Heavens exists but the sunder behaviour is D2R (U).
- Ammo: javelins are limited stacks that break/vanish at times; the bot must restock (V Kolbot Amazon.js buys arrows/bolts on repair; U equivalent for javelins).

## 5. Mercenary

- Act 2 Desert Mercenary, Might aura, Insight polearm for mana (V Starter) and Infinity later for Conviction (V Standard). Hire in every difficulty as soon as Lut Gholein is reachable (same rule as Sorceress plan, `docs/plan/data/mercs.json`).
- Act 5 Frenzy merc with Plague Phase Blades only for ubers (V), skip.
- Revival cost grows with merc level and never exceeds 50,000 gold (V Arreat Summit hirelings page https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/basics/hirelings.shtml; hire price is shown in the hire dialog, formula not found, U).

## 6. Dangers by difficulty

- Normal: Andariel poison (U), Fire Enchanted early; low life means ranged arrows are fine.
- Nightmare: first Lightning-immune packs and Cursed packs; start of the respec window.
- Hell: "Most Hell areas contain Lightning Immune monsters" (V). Fix before Infinity merc / Lower Resist wand: Jab, merc, or skip (V). Skip uniques immune to both lightning and physical (V). Dark Archers in the Pit use slow missiles (V). Physical immune: use Fire/Poison or Lightning, have 2 damage types.
- Class death causes (U): getting surrounded in melee range with Charged Strike; Fury needs mobs gathered but then they are on top of her; Valkyrie dies and aggro snaps back; running out of javelins mid-fight.
- Known common-to-all dangers (Mana Burn, Cursed, Lightning Enchanted etc.) are in `docs/plan/data/danger-table.json`.

## 7. Mana, potions, belt, charms

- Fury "has high mana costs" (V); Insight merc plus leech (V). Early Poison Javelin costs little (U). Belt policy as in `docs/plan/data/potions-belt.json` (health 2 columns, mana 1, rejuv 1), more mana potions than the Sorceress in Hell.
- Charms: Small Charms of Vita/life, resistance (Hell cap 75%, V generic), +1 skill-tree Amazon charm; Annihilus/Torch are unrealistic offline early (U).
- Also keep a spare javelin stack and a backup weapon (V: "carry spare javelins and a backup weapon").

## 8. What the bot needs (vs the Sorceress)

| Topic | Sorceress (current) | Amazon |
|---|---|---|
| Attack cast | Right-click `CastAt` ground/unit, no hold | Left or right skill on a monster; Fury/Poison Javelin `CastAt(unit)`; Strafe (rejected) would need hold-to-cast |
| Position check | Keep >= 10 subtiles, kite | Fury: get into skill range (V Kolbot `getIntoPosition(unit, range)`), keep the mob at range 12+ ; Charged Strike is melee, needs melee positioning |
| Summons | none | Valkyrie (recast on death) and Decoy: must be excluded from `Hostiles` (existing owner check covers summons: docs/combat.md) and recast on missing, needs an "owned unit count" read |
| Weapon swap | none | Teleport staff or Harmony bow on swap (V); combat needs `SwapWeapon()` |
| Ammo | none | quiver/javelin count read, restock in town, `goals` for buying ammo, repair also restocks (V Kolbot) |
| Damage-type fallback | cold -> lightning -> fire | lightning -> Jab/physical -> poison; add Jab as melee fallback needing melee positioning |
| Buff | Frozen Armor | none (no armor buff), Valkyrie/Decoy only |
| Existing behaviour to change | `Layout(build)`, `PickAttack`, kite distance | new Build enums (Javelin), an `AmmoLow` safety action, ranged-vs-melee skill range table, summon re-cast loop |

Estimated engineering delta over the Sorceress: medium (ammo, pets, two attack ranges).

# Druid plan (D2 LoD 1.14d, offline softcore, unattended)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** unverified (memory or judgement). Guide pages are Maxroll D2R Season 14; D2R-only things (Flame Rift sunder, cooldown tuning) are marked. Prefer the game's .txt tables at setup.

Sources: Maxroll Druid leveling https://maxroll.gg/d2/guides/druid-leveling (V), Fissure Druid https://maxroll.gg/d2/guides/fissure-druid (V), build list https://maxroll.gg/d2/build-guides (V; tier letters inferred, S), Maxroll immunities (V, D2R), Kolbot `Attacks/Druid.js` https://github.com/blizzhackers/kolbot (V).

## 1. Build choice

| Build | Safety | Mana | Gear | Immunities | Automation | Verdict |
|---|---|---|---|---|---|---|
| Fire: Firestorm / Molten Boulder / Fissure -> Fissure+Volcano+Armageddon | Medium; Maxroll calls the playstyle "dangerous", low mobility (V) | Poor-medium, Insight merc (V) | High (needs +skills, FCR 99% in the standard build, V) | Fire immune (V); Volcano/Boulder have a physical half | Ground-targeted AoE, 2 s Fissure cooldown (V) | **Primary** (matches Maxroll leveling route, V) |
| Tornado + Hurricane + Cyclone Armor | Medium: Hurricane cannot be cast in town, must be recast, Tornado misses (V) | OK | Medium | Physical immune skipped (V) | Tornado needs aim variance (V Kolbot randomises the x coordinate) | **Fallback** (Maxroll respec target at clvl 38, V) |
| Summoner (wolves, bear, vines) | Medium (pets tank) | OK | Low | None | Pets only; weak DPS | rejected (tier C, S) |
| Werewolf / Maul / Rabies | Low (melee, shapeshift) | | | | needs form-change | rejected |

Leveling build (Maxroll, V): Firestorm 1-5, Molten Boulder 6-7, Firestorm 8-11, Fissure 12-13, Fissure after, Firestorm from Radament/Izual; respec to Tornado/Hurricane/Cyclone Armor at 38 (V). For us: stay on Fissure/Volcano and skip the respec if gear allows; one build, no respec is simpler (U), cost is Fire immunes.

## 2. Level ranges, areas and runs

Maxroll bands (V): Act 1 Normal 1-13; Acts 2-5 Normal 14-26; Acts 1-3 Nightmare 27-38; Mephisto Nightmare to Act 5 Hell 39-75 after respec. Good farm spots for Fissure: Stony Tomb, Cow Level, Flayer Jungle (V). Bad: Maggot Lair and Arcane Sanctuary (corridors; use Firestorm/Boulder there, V).

## 3. Skill order and stats

- V Maxroll leveling order above. Respec allocation (V): Tornado 15, Hurricane 9, Cyclone Armor 14, optional Summon Grizzly 1; 5 left unallocated.
- Endgame fire (V): Fissure, Firestorm, Volcano, Molten Boulder, Armageddon, Oak Sage; wolves/ravens/grizzly are optional summons (up to 5 spirit wolves, 3 dire wolves, 1 bear, 5 ravens, V).
- Stats (V): 2-11 Vitality; 12-13 Strength; 14-21 Vitality; 22-26 mostly Strength (4-5 more at 26, or 5 if a Broad Sword for Spirit); 27-38 Vitality; 39-75 Strength for gear, rest Vitality.

## 4. Gear and runewords

Stealth (2-socket body), Leaf (Short Staff from Akara), Gemmed Helmet (Ral) and Lore helmet, Ancients' Pledge (Kite Shield from Farah), Staff of Teleportation from Ormus at clvl 20-23, Spirit on a Cow-Level Crystal/Broad Sword via the Act 5 Quest 1 socket, Nightmare resist belt/boots/gloves (V). Endgame (V): Heart of the Oak, Phoenix, Ravenlore, Flickering Flame helm, Stormshield; Flame Rift charm sunders fire immunity but is D2R-only.

## 5. Mercenary

Act 2 Desert Mercenary with a Pike from Halbu (V leveling, needed for Grandvizier of Chaos and Colenzo, V). Might aura and an Insight weapon early, Infinity polearm later for -85% fire resist (V). Same as Sorceress: hire every difficulty.

## 6. Dangers

- Normal: Andariel and Duriel melee if Fissure/Firestorm are not killing before contact.
- Nightmare/Hell: fire immune packs (skip uniques with fire and physical immunity, V); fire-resistant Pit Lords (resistance 70 in Normal to 120 in Hell, V); Mana Burn; Hell Ancients (circle and cast behind you, V); Baal ("hardest fight", V).
- Class death causes (U): Cyclone Armor absorbs only a fixed amount and must be recast (V recast rule in Kolbot Druid.js); ravens/wolves die and aggro shifts; Fissure leaves the caster standing at the front.

## 7. Mana, potions, charms

Mana is a limit (V says little); Insight (Meditation) polearm on the merc and Regenerate Mana gear (V). Belt: health plus mana columns, more mana than Sorceress. Charms: resists, Vita; Flame Rift (D2R).

## 8. What the bot needs (vs the Sorceress)

| Topic | Sorceress | Druid |
|---|---|---|
| Attack | right-click `CastAt` | right-click `CastAt` ground (Fissure/Firestorm/Volcano at a monster's position, cooldowns 0.6-4 s, V). Existing cast/delay loop applies |
| Buff | Frozen Armor, recast timer | Armageddon, Oak Sage, Cyclone Armor, Hurricane: state read for each (V Kolbot `me.getState(...)` Hurricane/CycloneArmor); Hurricane not in town; ravens/wolves recast by minion count |
| Summons | none | wolves, bear, raven, Oak Sage: pets owned by the player must be excluded from `Hostiles` (covered by the existing owner check, docs/combat.md) and counted for recast |
| Position | kite 10 subtiles | Fissure needs mobs to pass through the vents: cast ahead of the pack then retreat; Tornado needs lateral offsets |
| Skill fallback | cold->lightning->fire | Fissure -> Boulder -> Volcano -> Firestorm; physical half for fire immunes |
| Weapon swap | none | Teleport staff / Call to Arms on swap (V); needed for charge Teleport |
| Existing code to change | `Layout`, `PickAttack` | cooldown-aware `PickAttack` (it already respects Blizzard's delay), new Build enums, pet recast loop, ground-lead targeting |

Delta over the Sorceress: medium-high. Druid is the first class where summons, buff states and cooldowns all appear.

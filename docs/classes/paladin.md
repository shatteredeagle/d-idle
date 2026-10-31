# Paladin plan (D2 LoD 1.14d, offline softcore, unattended)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** unverified (memory or judgement). Guide pages are Maxroll D2R Season 14; D2R-only things (Renewed Black Cleft sunder, Patch 2.4 Unraveler change) are marked. Prefer the game's .txt tables at setup.

Sources: Maxroll Paladin leveling https://maxroll.gg/d2/guides/paladin-leveling (V), Maxroll Blessed Hammer Paladin https://maxroll.gg/d2/guides/blessed-hammer-paladin (V), Maxroll build list https://maxroll.gg/d2/build-guides (V), Maxroll immunities (V, D2R), Kolbot `Attacks/Paladin.js` https://github.com/blizzhackers/kolbot (V).

## 1. Build choice

| Build | Safety | Mana | Gear | Immunities | Automation | Verdict |
|---|---|---|---|---|---|---|
| Blessed Hammer (Hammerdin) + Concentration | High: Holy Shield block, Vigor, merc with Holy Freeze (V) | Insight merc Meditation, Redemption (V) | Medium: runes for Spirit/Enigma | Few; magic-immune need Smite/merc (V); hammers vanish on walls (V) | Cast `BlessedHammer` in a direction with Concentration on, no target needed; hammers spiral | **Primary** |
| Holy Fire + Zeal/Smite leveling (before respec) | Medium (melee) | Low need | Low | Holy Fire stops working after Act 3 due to fire resist (V) | Aura + melee: pathing must close to melee | **Leveling to 19** (Maxroll, V) |
| Fist of the Heavens (Foh) | Medium | | High | Holy bolt vs undead | Targeted; needs `Charge`/aura | rejected (gear heavy, S) |
| Smiter / Zealot | Low | | | | Melee | rejected |

Maxroll's plan (V): Might, Sacrifice, Holy Fire from 6-9, Smite at 10, Zeal/Charge at 12; respec at clvl 19 to Blessed Hammer (2), Charge (1), Concentration (1), Blessed Aim (8), Vigor (2); then Blessed Hammer to 36+, Vigor/Concentration/Blessed Aim after. Hammer needs clvl 24 as per skill (U).

## 2. Level ranges, areas and runs

Maxroll (V): Act 1 Normal to 13 (Den, elites, Countess for runes, Andariel); Act 2-3 to 19 (Farah belt, Radament, Cube re-roll of magic scepters for +Blessed Hammer, Stealth by 17, Ral gem cap); respec at 19; Act 4 to Act 1 Nightmare to 35 (Izual, Diablo, Act 5 quests, Cow Level for a Crystal Sword, Baal and throne waves); Nightmare Act 2 to Hell Act 5 to 76 (Countess in Nightmare for Spirit and Lore, Spirit Targe after Act 5 Quest 1). Buy a Staff of Teleportation from Ormus at clvl 20-23 (V). Hell XP: `progression-hell.json`.

## 3. Skill order and stats

- Pre-respec V: 2 Might, 3 Sacrifice, 4-5 Resist Fire (Den), 6-9 Holy Fire, 10 Smite, 11 save point, 12 Charge and Zeal, 13 Holy Fire; 14-19 Holy Fire, Resist Fire at Radament.
- Post-respec V: Blessed Hammer, Vigor on Izual, Concentration, Blessed Aim, then alternate Concentration and Vigor; keep synergy points about 12 above Concentration (V). 1 point each in Holy Shield, Cleansing, Redemption, Fanaticism (V endgame); Resist Lightning maxed (V).
- Stats V: 2-11 Vitality (1 Dexterity at 6), 12-13 Energy, 14-19 Vitality, 20-36 18 Strength then Vitality; Dexterity to 75% block with Holy Shield in the standard build (V); FCR 75% or 125%, FHR 86% (V).

## 4. Gear and runewords

Steel (optional, Tal+Eth scimitar), Stealth (Tal+Eth, clvl 17), Gemmed Cap (Ral) for fire resist, Lore helm, Spirit shield and sword from Tal/Thul/Ort/Amn plus Sol/Amn counts (V), Spirit Sacred Targe off-hand, Enigma body for Teleport, Hellwarden's Will / Harlequin / Crown of Ages helm, Call to Arms or Staff of Teleportation on swap (V). Re-roll a magic scepter with chipped gems for +Blessed Hammer (V). Runewords do not work in magic bases (V). Garnet belt/gloves and Ocher Boots of Haste for resists (V).

## 5. Mercenary

Act 2 Desert Mercenary with Holy Freeze (slow) and Insight for Meditation (V). Resist and life leech gear; Andariel's Visage + Fortitude in the standard build (V). Revival never costs more than 50,000 gold (V https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/basics/hirelings.shtml); hire fee is shown in the dialog (formula U); Act 2 mercs are the most party-friendly: their aura is shared with the whole party and minions (V same page). Potions can be fed by dragging to portrait (V); the bot should not rely on that.

## 6. Dangers

- Normal/Nightmare: Holy Fire's mana and melee early; magic immune "one monster that must be killed" and Unravelers (D2R patch 2.4 made them harder, V). Maggot Lair walls (V). Hammer needs open ground.
- Paladin auras reach party members within the aura radius (V Arreat Summit: 'Paladin Auras and Barbarian Warcries apply to party members, too', https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/basics/parties.shtml). Radius table (yards, V https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/skills/paladin-offense.shtml): Might and Concentration start at 10.6 yards at skill level 1 and grow about 1.3 per level. Concentration 'can also be shared with other party members' (V same page). Resist auras give party members and pets +max resist (V paladin-defense page).
- Hell: lightning (Resist Lightning maxed, V; Uber Mephisto Conviction); Lightning Enchanted / Mana Burn / Cursed; Souls and Dolls; physical-immune monsters ("few immunities", V).
- Class death causes (U): surrounded because hammers have a blind spot near walls; stuck in dead ends (V "weak in narrow areas"); Holy Shield expiry; auras dropped by swap; Cleansing missed for poison/curses.

## 7. Mana, potions, charms

Insight handles mana (V); Redemption restores life/mana from corpses (V); hammer cost scales with level (V); Smite costs 2 mana (V). Belt: 2-3 health, 1 mana, rejuv; Fewer mana potions than Sorceress. Charms: Vita, resist, +1 Paladin skills, Annihilus if found (U). FCR and resist gear are the main stat targets.

## 8. What the bot needs (vs the Sorceress)

| Topic | Sorceress | Paladin |
|---|---|---|
| Attack | right-click `CastAt(target)` | `BlessedHammer` cast at a point 1 tile in front and keep casting; Kolbot casts only when the target is within 9 tiles (V `unit.distance > 9`), uses `getHammerPosition` to find a free ground cell (V) |
| Hold-to-cast | no | hammers: repeated key press each cast (FCR frames); not a true hold; Zeal/Smite melee needs attack-hold (U) |
| Auras | none | Concentration while casting hammers, Vigor travelling, Holy Shield (cast, duration), Cleansing/Redemption interludes (V). Needs "set right skill" reliably: **aura switching = hotkey/skill-popup interaction with a ~5 frame delay** |
| Position | kite away | opposite of the Sorceress for melee leveling (Zeal/Smite, V); Hammerdin attacks from about 3-9 tiles and kites around a mob (V) |
| Weapon swap | none | Call to Arms or Staff of Teleportation swap, cast Battle Command x2 + Orders (V) on the swap; needed for Teleport charges |
| Teleport | native | no native teleport: Staff charges / Enigma; pathing to the target needs walk fallback; `nav` already walks |
| Immunity | element immunity table | physical immune is rare; Smite magic is the fallback; hammers are magic damage so physical immunity is irrelevant (U) |
| Existing code to change | `Layout`, `PickAttack`, kite | aura state machine (new), directional cast with wall check, melee leveling profile, Holy Shield recast timer |

Delta over the Sorceress: medium (aura switching and directional cast); smaller than the Barbarian and Necromancer.

## Live results: PALALIVE honest-play smoke (i2, Pally, 2026-10-29, no cheats)

Fresh clvl 1 Paladin played by `d2bot --attach --goals reach:7` in 4 background runs (about 10 min each). Tags: V live (job logs), U not seen.

- V start log: `class-aware start class=Paladin combatProfile=paladin levelerBuild=paladin-normal30` at clvl 1, 2, 4 (every start).
- V skill points follow `build-paladin-normal30.json`: clvl 2 Might (+2 Str, 3 Vit), clvl 3 Sacrifice, clvl 4 Resist Fire. Spending happens only in town on the next start/chore pass (points stay pending in the field after a level-up; harmless).
- V hotkeys: `combat hotkey ok Might f1`, `Resist Fire f7` (profile binds every known aura; Resist Fire is only a Holy Fire prerequisite, harmless). Left skill set by popup: `combat left skill set skill=96 (Sacrifice) cell (104,433)`; state read after: left 96, right 98.
- V auras: `combat aura up skill=Might tries=1` (ActiveAura via states reads the aura chain).
- V melee works with the start kit (Short Sword + Buckler, no weapon problem raised); `combat melee: backing off surrounded` fired in a Fallen pack; drinks at life 35-40 of 88; 0 deaths, clvl 4 after about 40 min of play (exp 5660 of 7500 for clvl 5), about 19k xp/h.
- V hireling/IsFriendly: no merc, no friendly-fire incidents in 400+ kills (U for a hired merc).
- Offline: `internal/gear/paladin_test.go` (start kit ok, bow and orb flagged).
- Final: clvl 4 after about 70 min over 4 runs (Den of Evil loop below cost much of it; clvl 6-8 not reached). Bug (not Paladin specific, owner: town/quest): Den of Evil "area cleared" fired with no boss kill (no `boss:true` kill in any run), so the Akara claim fails (`no skill point after talking to Akara`) and the planner revisits the Den. Same symptom noted in docs/container.md. Akara's first-talk speech takes about 2.5 min on screen before the claim returns.
- Slow: poverty mode is on from clvl 2 (0 gold, no TP tome), so no potion shopping; planner oscillates Blood Moor / Cold Plains at the border on "waypoint not usable".
- Bug: in the last run the claim trip also failed 5x with `town: no town portal available` (no TP tome, poverty) standing in the Den, escalating to failsafe ladder step 3; the claim trip should walk/waypoint home when no portal exists.

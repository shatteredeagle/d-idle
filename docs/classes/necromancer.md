# Necromancer plan (D2 LoD 1.14d, offline softcore, unattended)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** unverified (memory or judgement). Guide pages are Maxroll D2R Season 14; D2R-only things (Flame Rift, Bone Break, Annihilus sunder) are marked. Prefer the game's .txt tables at setup.

Sources: Maxroll Necromancer leveling https://maxroll.gg/d2/guides/necromancer-leveling (V), Summoner Necromancer https://maxroll.gg/d2/guides/summoner-necromancer-guide (V), Poison Nova Necromancer https://maxroll.gg/d2/guides/poison-nova-necromancer (V), build list https://maxroll.gg/d2/build-guides (V; tier letters inferred, S), Maxroll immunities (V, D2R), Kolbot `Attacks/Necromancer.js` https://github.com/blizzhackers/kolbot (V).

## 1. Build choice

| Build | Safety | Mana | Gear | Immunities | Automation | Verdict |
|---|---|---|---|---|---|---|
| Teeth + Bone Spear -> Bone/Poison (Maxroll leveling, V) | Medium-high: ranged bolt + Bone Armor/Prison | OK | Low | Magic/physical immune partly; Corpse Explosion covers (V) | Bolts are targeted `CastAt(unit)`; CE needs corpses | **Primary for leveling** |
| Summoner: Skeletons, Revive, Golem, curses | High (minions tank) but "uber and act bosses deal massively increased damage to summons" (V) | Not discussed (V) | Medium; steep positioning curve (V) | Handles physical immunes via Amplify Damage (V) | Needs summon upkeep and minion positioning (V: "placement accounts for more than half their power") | **Primary for Hell** (unattended, low risk) |
| Poison Nova | Medium | Poor | High (+skills, FCR 75/125%, V) | Poison immune (V) | Self-centred ring: needs mobs adjacent | rejected for early game |
| Bone Spear endgame | Medium | | | | | alternative (tier C, S) |

Maxroll's route (V): Teeth in Normal, add Corpse Explosion, respec at the start of Nightmare to Poison and Bone (Corpse Explosion 10, Bone Spear 9, Bone Prison 3). For the bot: since summoning needs corpses and many positioning tweaks, the leveling spec is Teeth+Bone Spear with Clay Golem; the endgame spec should be Summoner (V Maxroll build assumes character level 75 before switching).

## 2. Level ranges, areas and runs

Maxroll bands (V): Act 1 Normal to 13 (Den, Black Marsh, Forgotten Tower for Countess runes, Catacombs 4 for Andariel); Acts 2-5 to 26 (Radament, Cube, Staff of Teleportation at Ormus, Act 5 quests 1-2, Izual, Diablo, Cow Level, Throne waves); Nightmare Quest 1 and Countess for White Bone Wand and Lore runes (Dol, Io, Ort, Sol, V); then Hell via the Sorceress area list (`progression-hell.json`).

## 3. Skill order and stats

- V Maxroll: Teeth 2-5; Clay Golem 6; Teeth 7-10; Amplify Damage 11; Iron Maiden 12; Teeth 13; Bone Armor 14; Bone Wall 15-16; Corpse Explosion 17+; Bone Spear 18-20. Respec at Nightmare: CE 10, Bone Spear 9, Bone Prison 3.
- Summoner endgame (V): Skeletons, Revive, Clay/Iron Golem, Amplify Damage max uptime, Corpse Explosion, Terror, Life Tap for ubers; skeletons replace the oldest when the cap is hit (V).
- Stats (V): 2-3 Energy, 4-11 Vitality, 12-13 Strength, 14-16 Vitality, 17-21 22 Strength, 22-26 Vitality; later Strength for gear (25 for summoner standard, V), rest Vitality.

## 4. Gear and runewords

White Bone Wand (+Skeleton/Bone) from the Countess runes (V), +2/+3 Teeth wand from Akara, Bone Wand from Drognan before clvl 18 (+2-3 Bone Spear), Gemmed Helmet (Ral), Stealth, Ancients' Pledge, Lore, Rhyme for a Necromancer shield (V). Mid-late: Skin of the Vipermagi, Magefist, Stone of Jordan, Homunculus, Hsarus' Iron Heel (V). Teleport from Staff of Teleportation, Enigma (V). Annihilus sunder and Flame Rift/Bone Break are D2R; in 1.14d use Amplify Damage and the merc (U).

## 5. Mercenary

Act 2 Desert Mercenary: Might for hits on immunes; Infinity polearm later for Conviction (V). Equip resistances and leech; revival never exceeds 50,000 gold (V https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/basics/hirelings.shtml). Same hire rule as the Sorceress. Bramble Sacred Armor gives thorns to summons (V, endgame).

## 6. Dangers

- Normal: act bosses; curse Iron Maiden and keep a Clay Golem on the boss to return damage (V).
- Throne wave 2: Unravelers are magic immune (V): wall the spawn with Bone Wall, curses, Corpse Explosion on Skeleton Mage corpses. Pre-2.4 behaviour differs (D2R change).
- Hell: poison immune packs for Poison Nova (V); Physical immune ghosts with Stoneskin (V); Boss-class monsters deal heavy damage to minions, so summons will not survive Ubers/Act bosses (V). "Corpse Explosion damage does not scale with player count" (V), so it is weaker in a party.
- Class death causes (U): the necro's low life when minions die; teleporting into a pack with Revives stuck (V, move away after teleport); no corpses on boss fights.

## 7. Mana, potions, charms

Bone Armor costs 10 more mana than its level (V); Teleport falls to 1 mana at high levels (V); Insight on a merc or Iron Golem Meditation (V). Belt: mana-heavy, 2 health, 1 mana, 1 rejuv. Charms: Vita/Resist small charms, Annihilus (D2R sunder only).

## 8. What the bot needs (vs the Sorceress)

| Topic | Sorceress | Necromancer |
|---|---|---|
| Attack | right-click `CastAt` | Teeth/Bone Spear `CastAt(unit)` (V Kolbot range gating); CE needs a corpse target: `CastAt(corpse)` after reading dead monsters; Kolbot computes CE range as `(skill level + 7)/3` yards (V) |
| Curses | none | Amplify Damage / Lower Resist on elites, "only if curseable" (V `unit.curseable`); curse once per pack, reapply when the curse expires |
| Summons | none | Skeletons / Mages / Revive / Golem: counts via player-owned monsters, recast to the cap (V Kolbot `raiseArmy`), corpse validity states (V: frozen, shattered, revive, "bad corpse" list), no revive for some base ids (V) |
| Position | kite 10 subtiles | minions need space; stay behind the army; after teleport step back (V) |
| Buff | Frozen Armor | Bone Armor recast, Clay Golem recast (state/pool readable? U) |
| Weapon swap | none | Teleport staff; CTA on swap (V) |
| Existing code to change | `Hostiles`, `Layout`, `Buff` | owned-unit census (`mem`), corpse list from monster table (modes dead/ dying), curse cast, summon loop, hold-off when summons are low |

Delta over the Sorceress: high (corpses, summon cap, curses, minion micro).

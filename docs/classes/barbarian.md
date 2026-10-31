# Barbarian plan (D2 LoD 1.14d, offline softcore, unattended)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** unverified (memory or judgement). Guide pages are Maxroll D2R Season 14. Prefer the game's .txt tables at setup.

Sources: Maxroll Barbarian leveling https://maxroll.gg/d2/guides/barbarian-leveling (V), Whirlwind Barbarian https://maxroll.gg/d2/guides/whirlwind-barbarian-guide (V), Double Throw https://maxroll.gg/d2/guides/double-throw-barbarian-guide (V), build list https://maxroll.gg/d2/build-guides (V, tier letters are an inference, S), Maxroll immunities (V, D2R), Kolbot `Attacks/Barbarian.js` https://github.com/blizzhackers/kolbot (V).

## 1. Build choice

| Build | Safety | Mana | Gear | Immunities | Automation | Verdict |
|---|---|---|---|---|---|---|
| War Cry (leveling to 75, V) | Medium: War Cry is a stun area attack at melee range | Low need | Medium | Physical immune packs (V "some builds struggle to enter Hell") | War Cry is an area cast with no aim; positioning near mobs | **Primary** |
| Whirlwind | Low-medium: spin takes damage, no potions while spinning (V) | Mana leech key (V) | High | Physical immune (V: slow; needs Berserk/Bone Break/Decrepify merc) | Hold/drag movement skill, Kolbot has dedicated `Attack.whirlwind` (V); hardest to automate | rejected |
| Double Throw | Safe at range but "cannot move while using" (V) | OK | Very high (200-300 Dex, throwing weapons) | Physical (Bone Break D2R) | Straight line targeting | rejected |
| Frenzy / Berserk / Leap | melee | | | magic-based Berserk helps physical immunes (V) | | late alternate (Berserk) |

Maxroll's leveling route: melee with gas potions to clvl 31, then respec to War Cry + Battle Command + Battle Orders (V). Barbarian has almost no ranged attack, so every build is melee or short-line.

## 2. Level ranges, areas and runs

Per the Maxroll leveling page (V): Act 1 Normal to 15 (Den of Evil, Tristram/Cairn Stones, Countess for runes); Act 2 to 20 (Radament, belt and flails); Acts 3-4 to 25 (Flayer Jungle or Travincal, Khalim's Will as a temporary weapon to about clvl 24.7, then Mephisto, Izual, Hellforge); Act 5 to 31 (Eldritch until 29, Pindleskin, Shenk, Ancients, Baal); Cow Level for two Crystal Swords after respec; Nightmare Act 1 to 40; Nightmare Act 2 to Hell Act 5 to 75 with Baal runs. Hell spots in `progression-hell.json`.

Barbarian-specific: Find Item (hork corpses) is a Kolbot standard (V, `Barbarian.js` `findItem`), not needed for a leveling bot but a big loot gain and requires corpse targeting and a weapon swap (V).

## 3. Skill order and stats

- V Maxroll: 2-15 Bash, Mace Mastery, Double Swing (6-11); 16-25 Howl, Taunt, Leap Attack, Battle Cry, masteries, Increased Stamina, Shout, Double Throw, Frenzy, Battle Orders; 26-31 Bash then War Cry and Battle Command; respec at 31+ for War Cry max, then Battle Cry, Taunt, Howl.
- Stats (V): mostly Vitality; Dexterity 8-10 and Strength 11-13 early; at the respec 13 Strength, 40 Dexterity; later Dexterity for 50% block, rest Vitality.

## 4. Gear and runewords

Two scepters with gas potions early (V), Nadir in a 2-socket helmet, Stealth, Malice/Gemmed flail, Khalim's Will, Staff of Teleportation from Ormus (V), Spirit/Lore on Crystal Swords (V), resist belt/boots/gloves (Fire and Lightning first, V), Rhyme with a Bone Shield. Endgame: Death's Hand/Guard, Bloodfist, Arreat's Face, Bulwark, Stone of Jordan (V). Bone Break charm, Enigma teleport, Stormshield are D2R-era or endgame (V).

## 5. Mercenary

Maxroll hires an Act 1 Fire Mercenary (Rogue) with a Gemmed Hunter's Bow from clvl 20 (V). For a bot, the Act 2 Might Desert Mercenary (V for Whirlwind) is more robust and keeps the shared logic with the Sorceress plan. Reaper's Toll (Decrepify proc) breaks some physical immunes (V).

Party effect (V Arreat Summit https://web.archive.org/web/2020id_/http://classic.battle.net/diablo2exp/skills/barbarian-warcries.shtml): Battle Orders raises life, mana and stamina of party members, radius 2 yards at level 1 up to about 14.6 at level 20 (tables on the page); Battle Command passively gives party members +1 to skills for a limited duration and casting it twice also raises its own level; Shout/Battle Cry radii are on the same page. Party members must be within the radius (U that they must also be in the same area; aura radius cannot cross areas).

## 6. Dangers

- Normal: Andariel and Duriel melee; Flayer Shamans (heavy fire, V); Throne wave 5 (kite out of the big room, V).
- Nightmare/Hell: physical immunes; Mana Burn; Souls; Fire Enchanted; Conviction (cap resistances to 75% first, V generic); curse packs. All-melee means high hit rate on the bot: low safety.
- Specific death causes (U): potion lock during Whirlwind (V), stamina exhaustion when running (U), Stone Skin (Physical resist) elites, being swarmed by Hell ghosts (80% physical resist, V ghostly champions).

## 7. Mana, potions, charms

Barbarian needs almost no mana except Battle Orders/Command casts (V) and Find Item. Potions are the main sustain; mana leech helps (V) and Battle Orders gives life, mana, stamina. Belt: three health columns, one rejuv. Charms: life/resist small charms, Bone Break (D2R).

## 8. What the bot needs (vs the Sorceress)

| Topic | Sorceress (current) | Barbarian |
|---|---|---|
| Attack | right-click `CastAt` from range | melee: left-click-attack or War Cry cast near mobs; Whirlwind needs hold + mouse path (rejected) |
| Position | `kiteDistance` 10 subtiles, never melee | **opposite**: close the gap, stand in melee; today's `approachPoint`/`awayPoint` assume ranged; need a Melee profile with an "engage" distance and a surround check |
| Buffs | Frozen Armor | Battle Command x2, Battle Orders, Shout cast on a weapon swap with +warcries (V); the swap weapon is a required item swap |
| Weapon swap | none | `SwapWeapon` primary/secondary; also Find Item swap (V Kolbot) |
| Stamina | none | stamina belt/regen, `run` toggling, Increased Stamina; read stamina stat |
| Immunity | element immunity table | physical immune is the problem: need Berserk (magic) point, merc, skip; needs `monster.resist[phys]` read |
| Potions | belt hotkeys 1-4 | same; no potion during whirl (avoid WW) |
| Existing code to change | `Layout`, `PickAttack`, `kite*` tuning | melee target ranking (not "kill from range"), stamina knob, buff cast on weapon swap, physical-resist read |

Delta over the Sorceress: high (inverts positioning and survival assumptions).

## Live results (agent BARBLIVE, 2026-10-29, instance i3, Barby, no cheats)

Tags: V = seen live this session, U = not reached.

- Build: `barbarian_melee_normal30` (internal/gear/builds, copies in internal/plandata and docs/plan/data; `gear.DefaultBuild("barbarian")`, `leveler.buildFiles`). Order: Bash 2-4, Axe Mastery x2 at 5 (Den point), Double Swing 6, Howl 7, Shout 8, Double Swing 9-10, masteries, Double Throw 12 (only the Frenzy prerequisite), Increased Stamina x2 at 15, Iron Skin 18-19, Frenzy 24-30, Battle Orders 26-27; stats Vit first, Dex 8-10, Str 11-13 (rows are U, the guide gives ranges; prerequisites and reqlevels are checked by gen_builds.py). `TestBuildForClass` no longer treats Barbarian as the no-build case.
- V fresh Barbarian: 55 life, 10 mana, 92 stamina, str 30 dex 20 vit 25 ene 10, 0 skill points, left/right skill 0, body 4 item class 0 (Hand Axe), body 5 class 328 (a shield, name U), 4 minor healing potions, TP and ID scrolls. Axe Mastery is therefore the sensible starting mastery (the guide's Mace Mastery would idle).
- V `d2bot --attach --goals reach:8 --autostart`: class-aware start picked profile `barbarian` and build `barbarian_melee_normal30`; the leveler spent level 3 (+5 vitality, +1 Bash), level 4 (same) and level 5 (+5 vitality, +1 Axe Mastery) in the field with `statLeft 0 skillLeft 0`; the Bash left-skill popup cell (104,433) worked on the first try (SetLeft, V skill id 126).
- V Barby went clvl 1 to 5 (exp 13674 at the end, clvl 6 not reached) in about 85 minutes over three runs of Blood Moor loops, 0 deaths, no chicken exit; about 17 minutes per level at clvl 3-5 on a loaded worker host (fresh-game cycle each area clear).
- V bug fixed: with `manaReserve 0` the fighter flipped Bash <-> Attack every few seconds (Bash costs 2 of 10 mana, each switch is a popup round trip of about 1.5 s). `manaReserve` is now 4 in the barbarian profile (same hysteresis as the Paladin); the flips dropped from about 5 to 3 per two minutes, still visible because the pool is only 10-13 mana at clvl 1-5 (U whether Attack alone would be faster below clvl 6).
- U not reached: Double Swing (clvl 6), Shout, Iron Skin, Frenzy, Battle Orders casts, `Buffs` for the barbarian profile (a buff cast needs a learned skill; first one in the build is Shout at 8), Increased Stamina, stamina handling when running, dual-wield Frenzy behavior.

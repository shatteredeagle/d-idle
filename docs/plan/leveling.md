# Sorceress leveling and strategy plan (D2 LoD 1.14d, offline, unattended)

Evidence tags: **V** fetched page, **S** search snippet, **R** recollection or design judgement (unverified; check against the game's own .txt tables at setup). Many fetches were blocked (403), so most numeric tables are R. The data files in `docs/plan/data/` are the machine-readable form; the bot should prefer values read from the game data (levels.txt, skills.txt, monstats.txt, hireling.txt) over any number here.

Sources: Maxroll Sorceress leveling guide (V), Maxroll Blizzard Sorceress (V; written for D2R, mechanics are the same in 1.14d except D2R-only items and runes), DiabloBytes area levels (V), PureDiablo/Diablo Wiki area level snippets (S), Icy Veins and D2DB snippets (S).

## 1. Build choice

| Candidate | Ranged safety | Mana | Merc synergy | Immunities | Automation ease | Verdict |
|---|---|---|---|---|---|---|
| Blizzard + Static Field (+Glacial Spike) | High (cast at distance, ground target) | OK with Spirit/Insight | Good (Might merc covers cold immunes) | Cold immune (~20% of Hell packs) is the weakness; Static Field + merc | Ground-targeted AoE, one point target per pack, easy kiting with Teleport | **Primary** |
| Lightning + Chain Lightning (Lightning Mastery) | High, needs line of fire | Poor-medium | Fine | Lightning immune is rarer than cold | Chain Lightning targets a monster, easy | **Fallback** (no respec needed) |
| Frozen Orb/Fire Ball/Meteorb | Medium | Poor | Fine | Needs 2 elements | Orb needs aim; Meteorb needs expensive gear (Spirit Ward/ Griffon) to shine | rejected |
| Nova leveling (Maxroll V) | Low (point blank) | Poor | n/a | | Needs melee-range positioning | rejected for unattended play |

Reasons for the choice: Blizzard needs only a ground point, the cast delay is irrelevant if the bot keeps distance, Teleport at 18 removes most pathing and stuck problems, and the pre-respec phase (Charged Bolt, auto-targeting, no aim) is the simplest attack to automate. Maxroll (V) respecs at clvl 26 with Akara's free respec; the trigger quest is R. If no respec is available the bot switches to the Lightning/Chain Lightning fallback from clvl 26 without wasting the Charged Bolt/Static/Teleport points. Both use the same points to clvl 25 (`build-*.json`).

Class pluggability: the data files keep build-specific content (skills, hotkeys, pickit class tweaks) separate from the engine rules.

## 2. Allocation, belt, hotkeys

Stats (all builds): Dexterity 0, Energy 0. Strength only to the next gear requirement, rest Vitality (V Maxroll: "enough Strength to equip gear, rest Vitality"). Strength targets: 22 by clvl 11-17 (V Maxroll: 22 Str at clvl 17), 30 at 25, 45 at 30, 60 at 40, 75 at 55, 90 at 75 (R, adjust with gear requirement). Lam Esen's Tome gives +5 unassigned stats (R) after Normal Act 3.

Full 2..99 tables: `data/build-primary-blizzard.json`, `data/build-fallback-lightning.json`.

Skill order summary (primary):
- 2-5 Charged Bolt (Den of Evil quest point goes to Charged Bolt too, V), 6 Static Field, 10 Telekinesis (Teleport prerequisite, R verify prereqs in skills.txt), 18 Teleport, 22 Frozen Armor, others Charged Bolt.
- Save to clvl 26, respec at Akara: Static Field 5, Telekinesis 1, Teleport 1, Frozen Armor 1, Ice Bolt 1, Ice Blast 7, Glacial Spike 8, Blizzard 3 (27 points; Maxroll V). Then Blizzard to 20, Cold Mastery at 30 and onward (V: "Blizzard + Cold Mastery at 30"), Glacial Spike, Warmth 1.
- Main attack switches: Charged Bolt 1-25, Blizzard from the respec, plus Glacial Spike opener on melee threats from clvl 44 (V Maxroll gets GS at 44).

Static Field: open every elite/champion/boss pack with 3-4 casts at range (R: reduces life to 50% in Normal and ~25%? only to a floor: 25% Normal, 33% NM? Hell 50%? floor values R). Maxroll (V): start with Static Field on elite groups. Against Cold immune monsters Static Field plus merc is the damage plan (V).

Potions and belt by level: `data/potions-belt.json`. Thresholds: health potion at 55%, rejuv at 35%, chicken at 25% (30% in Hell), mana at 25%.

Hotkeys: `data/hotkeys.json`. Left = main attack, right = Teleport (from 18) else Static Field.

Teleport at 18: the bot can hop to any walkable tile within ~ screen distance, ignoring monsters and most terrain, so navigation changes from A* walking to "pick a chain of teleport hops along the path, reachable, within range". Cost ~24 mana (R) and a short cast delay, so mana and FCR matter. Needs a walkable destination from the collision map, not just a straight line. Keep walking for doors and quest NPC clicks and for towns (town teleport is allowed).

## 3. Mercenaries

Details in `data/mercs.json` (all R: verify auras from the hire dialog).
- Act 1 Rogue (Normal) only as a stopgap before Act 2.
- Act 2 Desert Mercenary from Greiz in Lut Gholein for all three difficulties: Normal Defensive aura (Prayer/Defiance), Nightmare Holy Freeze or Might, Hell Might (V Maxroll Blizzard build uses a Might merc against cold immunes). Gear: Insight polearm at clvl 27 (V), a helm and armor from drops, Infinity polearm at clvl 63+ (V Maxroll set build).
- Hire each difficulty as soon as the Act 2 town is reachable; revive in town after every death (NPC Greiz dialog), farm gold first if short.

## 4. Area and quest progression

Area ids (levels.txt; R, verify at setup): `data/areas.json`. Waypoints by act and per-step level windows: `data/progression-{normal,nightmare,hell}.json`.

Known area levels (V): Blood Moor 1/36/67, Cold Plains 2/36/68, Stony Field 4/37/68, Dark Wood 5/38/68, Black Marsh 6/38/69, Tristram 6/39/76, Spider Forest (A3) 21/49/79, Outer Steppes (A4) 26/56/82, Frigid Highlands 25/59/81, WSK 2 40/65/85, Colossal Summit not in plan. In NM and Hell regular monster level = area level, champions +2, uniques +3 (V). 13 alvl-85 zones in Hell (Pit 1/2, Ancient Tunnels, Chaos Sanctuary, etc., V DiabloBytes, but note that is the D2R 2.4 list; in 1.14d the set differs: do not rely on it).

Normal (target clvl at end: 38-41 by Baal; Maxroll V: ~19.8 at end of Act 3 on its path, 26 by Throne):
1. Blood Moor, Den of Evil (clvl 2-4; quest worth it: 1 skill point, V), Cold Plains, Burial Grounds (Blood Raven).
2. Black Marsh WP, Forgotten Tower to Tower Cellar 5 (Countess; Maxroll V: for runes), Outer Cloister WP, Catacombs 4 Andariel (clvl >= 14-22).
3. Act 2 by ~17: Radament (+1 skill point), Cube, Staff, Amulet; Summoner, Duriel (clvl >= 26 comfortable).
4. Act 3: Lam Esen (+5 stats), Khalim's Will chain, Travincal (range only), Mephisto.
5. Act 4: Izual (+2 skill points), Diablo. Act 5: Larzuk (sockets), Anya (resistance scroll), Ancients (skip until 28+), Baal.
Skip: Cairn Stones/Tristram is optional but quick for Wirt's Leg (needed for Cows).

Nightmare (enter at clvl 34-36 min; Maxroll HC 38-40): same order by act, Andariel at 45, Duriel at 50, Mephisto 55, Diablo 55, Baal 58. Kill Lord de Seis first in Chaos Sanctuary because he may be cold immune (V). Ancients: skip Korlic's pack variants (V).

Hell (enter at clvl 65; resist cap 75 before entering, V Maxroll): Skip Den of Evil (V). Use Teleport past cold immune packs (V). Boss min clvl: Andariel 65, Summoner 70, Mephisto 75, Diablo 80, Baal 82 (R).

Golden path once geared (clvl 80+, R): Countess (runes), Pindleskin (Red Portal at Harrogath, area 121), Eldritch + Shenk, Andariel, Mephisto, Baal, Cows (clvl 70+ to Cow King off), Pit, Ancient Tunnels. `data/runs.json`.

XP rates (R; rough): Normal Act 1 2-3 levels/hr to clvl 12, then ~1/hr, clvl 20 around 4-6 h total; Nightmare 30->50 ~15-25 h; Hell 65->85 ~40-70 h with Baal/Pindle/Cows runs. Idle schedule: day 1 Normal to clvl 26 (4-6 h), day 2 NM 26-45, day 3-4 NM to Hell entry 65, day 5-12 Hell to 85, then runs for loot. Treat all as R: the bot should measure XP/hour per area and adapt (below-expectation -> change area).

## 5. Danger rules

`data/danger-table.json`. Principles:
- Never melee. Keep >= 8 tiles from non-boss monsters; Teleport away from anything that closes under 4 tiles when life < 70%.
- Kill first: Fallen/Carver Shamans, curse casters, aura-enchanted (Fanaticism).
- Avoid: Mana Burn and Lightning Enchanted in Hell until clvl 75, Cursed in Hell (Lower Resist), Conviction (resists removed), Fire Enchanted (kill at range), Dolls (explode), Stranglers, Souls (mana drain), Thorns packs (merc dies).
- Immunity: read monster resistance stat via stat list (39 fire, 41 lightning, 43 cold, 45 poison; >= 100 immune). Blizzard immune -> Static Field + merc + stun; elites cold immune -> skip. Never spend >45 s per pack.
- Potions and chicken: drink health at 55%, rejuv at 35%, leave game (chicken) at 25% (30% in Hell). Town portal: cast TP scroll when 2 potions (health) or less, belt column empty, or merc dead and wanting revive (not mid-fight).
- Death: offline single-player, corpse stays in the game where you died. If dead in town or in a known-safe place, retrieve; otherwise rejoin via the character select: the player respawns in town naked of equipped items? (the corpse holds the equipped gear). Rule: retrieve corpse only if the run was <= clvl-appropriate and the area is cleared or low-risk (Normal Act 1-2); otherwise abandon only if the corpse has no key items, else make a new game and teleport-run to the body. Expect 1-2 deaths per 100 hours at Hell. Equipped items stay with the corpse (R).

## 6. Town loop and loot

Town loop (each trip): heal at NPC (free via fountain of life? no; use Akara/Malah), buy potions to belt targets, repair (Charsi/Hratli/Halbu/Larzuk), identify (Cain scroll or Deckard Cain), sell B-tier items, stash S/A runes and gems and charms, keep gold < stash cap 2.5m and use at Hell for Atma/Fara/Halbu items, buy TP/ID scrolls (Tome of TP, Tome of ID: buy a Tome at Akara/Drognan). Gold gamble: at clvl >= 30, gamble rings/amulets/circlets when gold > 100k (R), keep 20k reserve; nothing before.

Loot: `data/pickit-starter.json`; Runeword targets and bases: `data/runewords.json`. Plan: Stealth (Tal+Eth) at 17, Ancients' Pledge at 19, Spirit at 25, Lore, Insight for the merc at 27 (V Maxroll starter core: Spirit, Stealth, Lore, Ancients' Pledge). Enigma (Jah Ith Ber) is the end goal (R), feasible from Hell Mephisto/Baal/Cows runs.

## 7. Time/XP estimate and schedule

See section 4. Recommended schedule: run 4-8 h sessions, restart the game every ~2 h, rotate difficulty goals as in `progression-*.json`, stop at Hell clvl 85 and then farm runs.

## Uncertainty
Area ids/auras/quest points and all level thresholds are R. The first engineering milestone should be to dump levels.txt, skills.txt, hireling.txt, monstats.txt, superuniques.txt from the user's MPQs and diff against these data files.

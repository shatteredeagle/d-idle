# Ranking: how well can an unattended bot play each class (offline softcore, 1.14d)

Tags: **(V)** fetched this session, **(S)** snippet/secondary, **(U)** judgement or memory. The scores below are our judgement (U) built on the per-class files in this folder; every score cites the class file section it comes from. Sources are the class files' source lists plus Maxroll build list https://maxroll.gg/d2/build-guides (V; its tier letters are an inference, S) and Kolbot attack scripts https://github.com/blizzhackers/kolbot (V).

## Criteria (1 = bad for a bot, 5 = good)

| Criterion | What a 5 means |
|---|---|
| Safety | Ranged or tanky, few ways to die unattended; survives a missed potion tick |
| Speed to Hell | Reaches Hell clvl ~64 with few slow phases; has teleport early (V: Staff of Teleportation from Ormus at clvl 20-23 for any class) |
| Gear independence | Works with crafted/runeword/vendor gear, not unique/set |
| Input simplicity | One or two targeted casts, no combos, no hold-to-cast, little buff/pet upkeep |
| Mana | Mana is not the limiter; potions and Insight merc suffice |
| Immunity handling | Many damage types or an automatic answer to immunes |

## Matrix

| Class (build) | Safety | Speed | Gear | Input | Mana | Immunity | Total |
|---|---|---|---|---|---|---|---|
| Sorceress (Blizzard + Static, Teleport 18) | 4 | 5 | 3 | 5 | 3 | 2 | **22** |
| Paladin (Hammerdin) | 4 | 3 | 4 | 3 | 4 | 4 | **22** |
| Necromancer (Bone Spear -> Summoner) | 4 | 2 | 3 | 2 | 3 | 3 | **17** |
| Assassin (Lightning Sentry traps) | 3 | 3 | 3 | 3 | 3 | 2 | **17** |
| Amazon (Javelin / Lightning Fury) | 3 | 2 | 2 | 4 | 3 | 2 | **16** |
| Barbarian (War Cry) | 2 | 3 | 2 | 2 | 5 | 2 | **16** |
| Druid (Fire / Fissure) | 3 | 2 | 2 | 2 | 3 | 3 | **15** |

Short reasons (all U unless tagged):
- Sorceress: ground-target Blizzard and point-targeted bolts, Teleport removes pathing risk (`docs/plan/leveling.md`); weak spot is cold-immune Hell packs.
- Paladin: highest safety in guides (Holy Shield, Vigor, Holy Freeze merc, V) and "few immunities" (V) since hammers are magic damage; but no native Teleport (Enigma or Staff charges, V) and aura switching.
- Necromancer: minions soak damage and curses fix physical immunity (V Maxroll summoner), but corpses, summon cap, curses and minion positioning are costly input work and ubers one-shot minions (V).
- Assassin: remote traps are safe (V "even off-screen"); 5-trap cap and lightning immunity solved by merc/Lower Resist (V); trap planner needed; martial arts excluded.
- Amazon: easy targeted casts but "among the slowest", lightning immune "most Hell areas" (V), ammo and pets to manage.
- Barbarian: inverts our ranged design and has physical immunes (V), potion lock while whirling (V). Mana needs are the lowest of the seven.
- Druid: Fire builds depend on good gear and cooldowns, Hurricane/armor upkeep, summons; Maxroll calls the fire playstyle "dangerous" (V).

## Sensitivity

- If the bot runs mostly in parties (docs/net/multiplayer.md), rank Necromancer and Paladin up (curse/aura support, see `party.md`).
- If gear is poor (poverty mode after deaths, PLAN "Death is normal"), Paladin and Sorceress stay best; Amazon/Druid drop further.
- If we accept no respec and a single build, Druid and Necromancer lose most.

## Recommendation: build the Paladin (Hammerdin) second

Reasons:
1. Matches the Sorceress in safety score but fails differently (few elemental immunities vs her cold weakness), so two characters diversify risk.
2. Paladin leveling (Maxroll V: Holy Fire then respec at 19 to Hammer) is a short, well-documented script; Kolbot's `Paladin.js` shows the attack logic (V): directional hammer cast, 9-tile gate, wall-aware position finding.
3. Auras are the cheapest party synergy (Concentration, Vigor, Fanaticism, Holy Shield) and need no orchestration beyond staying in range (U; see `party.md`).
4. Engineering delta is the smallest of the six (class file section 8): aura state machine, directional cast, Holy Shield timer, weapon swap; no pets, ammo, corpses or traps.
5. Risk: no native Teleport; needs Staff of Teleportation charges (V Ormus buy) or Enigma, so `nav` walking must stay solid first.

Runner-up: Necromancer, as the party-support class (curses plus summons), second-highest value once summon upkeep exists.
Not recommended first: Barbarian (needs the largest combat rewrite).

## Open questions
- RESOLVED: charged items recharge when the town blacksmith repairs them (V Arreat Summit chargedsuffixes page); exact staff charges/level from Ormus still U.
- Are hammer and bolt casts rate-limited by FCR frames enough to need timing code? (needs live test, U)
- Mercenary revival never exceeds 50,000 gold (V Arreat hirelings page); hire fee formula not found (U).
- Real 1.14d numbers for monster resists per difficulty: read `monstats.txt`, not these docs.

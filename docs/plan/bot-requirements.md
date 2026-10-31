# Bot requirements implied by docs/plan/leveling.md

Checklist for engineers. D = game datum, C = capability. Every item should be marked covered/not covered by the owning agent.

## Game data (from MPQs / memory)
- D1 Tables: levels.txt (area ids, MonLvl/Normal/NM/Hell, waypoint flags), skills.txt (prereqs, mana, cast delay, range, synergy), monstats.txt (resistances, immunities, ai, ranged/melee, boss flags), superuniques.txt (Andariel, Mephisto, Baal, Eldritch, Shenk, Pindleskin, Countess ids and areas), hireling.txt (aura per hire id), armor/weapons/misc.txt, itemtypes.txt, runes.txt (Runeword.txt), sets/uniques, charstats.txt.
- D2 Player: class, level, xp, difficulty, current area id, position, life/mana/stamina (max and current), stats (str/dex/vit/ene, unspent stat points, unspent skill points), resistances, FCR/FHR.
- D3 Skills: learned skills with levels, hotkey bindings, current left/right skill.
- D4 Quests: per-quest flags per difficulty (Den, Radament, Lam Esen, Izual, Anya scroll, Larzuk), waypoint-activated flags per act.
- D5 Items: inventory, stash, belt (4x4), cube grid, equipment, item quality/ethereal/sockets/identify state/runeword, item stats; ground items with name/quality (need name hover or item unit read).
- D6 Monsters: unit id, class id, hp%, position, state, monster flags (champion/unique/minion), affix/enchant mods, auras, resistances from stat list.
- D7 Objects: waypoints, stash, portals, doors, chests, shrines, quest items; preset units of the level.
- D8 NPCs: vendor ids per act and buy/sell UI state; mercenary state (alive/dead, level, aura).
- D9 UI state: panel open flags, dialog state, game vs menu, difficulty selector, character list.

## Capabilities
- C1 Walkable grid and A* nav with doors; Teleport hop planner (range, line of walkable tiles, mana).
- C2 Waypoint travel and TP-scroll town return; area-id based goal-reaching.
- C3 Combat: target selection, cast ground-targeted spells (Blizzard) and unit-targeted (Charged Bolt, Chain Lightning), Static Field opener, kiting, per-target timeout.
- C4 Survival: potion thresholds, chicken (exit game cleanly), TP escape, merc potion, belt maintenance.
- C5 Immunity detection and switch/skip logic.
- C6 Danger affix checks by difficulty (danger-table.json), including mana-burn detection via mod list.
- C7 Level-up executor: spend stat and skill points from build-*.json; verify prereqs; respec flow at Akara; quest-point handling.
- C8 Town loop: heal, buy potions/scrolls, repair, identify, stash, sell, gamble; NPC dialog via UI with verification.
- C9 Loot: pickit evaluator (pickit-starter.json), item read (needs identified stats), pickup by walk/click, inventory space mgmt.
- C10 Cube: runeword crafting (insert runes into sockets, rune upgrade recipe), socketing via Larzuk, item swap/equip by UI.
- C11 Mercenary: hire, equip, revive.
- C12 Quest handlers: Den of Evil, Radament, Cube, Staff/Amulet, Lam Esen, Khalim's Will, Izual, Larzuk, Anya, with per-difficulty completion detection.
- C13 Run scripts: runs.json (area chain, boss fight, loot, return), per-difficulty min clvl gate.
- C14 Death/corpse handling, game restart/rejoin, OOG menu (character select/ create game, difficulty).
- C15 Progress metrics: XP/hour per area, loot logs, death counts; goal manager integration (clvl goals, run goals).
- C16 Safe-guard timers: stuck detection, per-area time cap, abort on unexpected panel, freeze detection, max session length.

## Open verification list (V needed)
Area ids (areas.json), waypoint area list, hireling auras per difficulty, Akara respec trigger, Teleport prereqs and mana cost, Static Field life floor per difficulty, exact superunique areas for Eldritch/Shenk, corpse behavior on death offline.

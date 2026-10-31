# Lessons: what to adopt, what breaks, where we differ

## A. Fifteen ideas to adopt
1. **Hover-confirmed clicking** (TY, KO): never click an entity until the memory hover buffer says the cursor is over that unit id/type; search around the projected point with a spiral (up to ~45 attempts for items, ~5 for monsters). Fail closed, never blind-click. (03, 05, 06)
2. **Preset-driven navigation, not coordinates**: find objects/monsters/exits by preset id (Countess chest x selects the stand point, Duriel tomb, Chaos seal A/B layouts) and keep a few verified fixed coordinates as fallback. (02)
3. **Data-driven builds**: stat/skill order as tables with level-based AutoBuild config rewriting (SP). Choose build by level, not by stored flag, so state can't desync. (01)
4. **Script skeleton with skip/need predicates** (`preReq/skipIf/shouldRun`) and a killed-set so a restarted game does not redo bosses. (01, 02)
5. **Watcher loop independent of the behavior loop**: potions, chicken, merc, ping at ~20-100 ms cadence with cooldowns (heal 1 s, rejuv 0.3 s, merc rejuv 2 s). (07)
6. **Town trip instead of quit** when low on hp/potions, with a guard against recursion and a 4.5 s settle before returning. (07)
7. **Order of town chores and need-checks** so the bot only visits NPCs it needs; choose nearest NPC when only selling. (05)
8. **Belt model**: columns typed hp/hp/mp/rv, min-per-column refill threshold, shift-buy, clear wrong potions first. (05)
9. **Target sort with "support first"** (shamans, Oblivion Knights, unravelers) and per-monster attack counters (drop after 15 attacks, requeue on position failure). (04)
10. **Timed/untimed skill pair** and immunity fallback skills; skip monsters immune to everything. (04)
11. **Stuck ladder**: 20-sample position history (KO), blacklist last cell + repath, perpendicular side-step, random click, then fail the step. (03)
12. **Teleport spacing vs walk spacing and layered fallback** (teleport -> walk when mana short; shrink spacing on retry). (03)
13. **Layout fingerprints** (TY) as a safety gate for any recorded/fixed data: hash semantic anchors, refuse to run on mismatch. Use for towns and Chaos Sanctuary. (03)
14. **Explicit budgets and an escalation ladder** (retry, repair, abort script, town, leave game, restart client, alert). (07)
15. **Handicap-ready parameters**: everything above has a number (ranges, delays, thresholds, retry counts); put each into `Tuning` as PLAN requires. Kolbot's `Config.*` keys are a ready catalogue: UseHP/UseRejuvHP/UseMP, LifeChicken, TownHP/TownMP, HealHP, PickRange, MaxAttackCount, Dodge/DodgeRange/DodgeHP, ClearType, SkipImmune/Enchant/Aura, StashGold, LowGold, MinGameTime. (all docs)

## B. Ten most common failure causes to design against
1. Panels or items-on-cursor swallowing clicks (inventory, NPC menu, waypoint, stash, cube): clear UI flags and cursor before every movement/click sequence.
2. Lost references: units die, get raised, vanish from the table, or are not yet loaded; always re-resolve by id and handle null.
3. Clicks that don't move the character (walls, doors, barrels, off-screen targets, UI region at screen bottom): stuck detection plus alternate click points; clamp mouse Y above the HUD (TY: <= 74 percent of height).
4. Portal races: using a portal too soon (2.5 s), wrong owner, stale portal object, dummy name.
5. Waypoint UI/area-change timing: wait for the area id, not for a fixed time; loading screens return invalid state.
6. Collision/map not yet loaded for the target (unexplored or non-floor areas like Arcane, Chaos, River of Flame, Abaddon, Pit): snap targets to nearest walkable; use soft costs near walls.
7. Boss ranged attacks and packs at exits (Mephisto, Andariel poison, Duriel, seal elites, Baal waves): positional rules plus clearing before interacting.
8. Out of resources mid-run (potions, TP scrolls, mana, merc dead): explicit checks with town trip; never enter a fight without a TP in the tome.
9. Inventory full or item does not fit: free space by selling/stash via town, mark item ignored; check fit before walking to the item.
10. Infinite loops from no-progress states (no damage, immune pack, unreachable item, chest unreachable): per-target counters, no-progress timers, per-run cap.
Extra: lag/desync, spawn variants (town exit layout, seal layouts), potion cooldown spam, merc dead after area change (null for a moment).

## C. Where our design (memory read, window-message input, no function calls) forces different solutions
| topic | in-process bots (D2BS) | ours |
|---|---|---|
| map/path | `getPath`, `getRoom`, `getCollision` for the whole level, even unexplored | `d2map` reads only loaded rooms; need our own A*, an explorer, and possibly map generation via d2mapapi (memory doc 3.3) |
| movement | `clickMap` on world coordinates; `moveTo` packets; `Packet.teleport(x,y)` | PostMessage mouse clicks at screen points via iso transform (PLAN: wx-wy = dx/16, wx+wy = dy/8), mouse position must be verified by hover; no packets, no teleport packet: teleport = right-click cast |
| item/NPC interaction | `Packet.entityInteract(unit)`, `Packet.buyItem`, `itemEvent` | physical clicks only, hover verify, buy/sell via shop UI click coordinates; NPC menus via text-line clicks |
| UI | `getUIFlag`, `me.cancel()`, `getControl()` | UI flags in memory (2.11); `launch` walks the control list from FirstControl; Esc key to cancel |
| skills | `me.setSkill`, `Skill.cast` | press the skill hotkey and verify selected skill in memory (2.8); skills need hotkeys bound in the game's skill bar (preparation step) |
| stats/points | `me.getStat` | stat list parsing; allocate points via stat/skill windows (click +) |
| events | `gamepacket`, `itemaction`, `gameevent` listeners (Diablo "lights", Baal waves, drop events) | poll memory each tick and diff (new items, monsters appearing, area change) |
| timing | host `delay`, ping known | our poll rate (tens of ms), `max(ms, 2*ping)` irrelevant offline; use frame/tick measurement of action animation to avoid spamming clicks |
| quest/WP data | `getQuest`, `getWaypoint` | memory locations unverified: find quest bit array and WP bitmask; fallback infer from area access |
| merc hire list | merc packet | read from shop/NPC UI memory or screen text |
| `quit()` | instant | UI Esc -> Save and Exit; exiting needs its own state machine |
| anti-detection | irrelevant (offline) | irrelevant, but action jitter is part of `Tuning` handicap |
Other implications: (a) hover buffer + mouse position are the only "pointing" feedback; build every interaction as `moveMouse -> verify hover -> click -> verify effect`. (b) The client animates actions; two clicks too fast are dropped. (c) No `Packet.flash` re-sync: use a click on own position/stop-move click, or Esc. (d) 800x600 window means a small viewport (about 25 x 25 tiles visible): targets beyond ~20 units need a move first. (e) Shift-click "attack in place" and the Alt key to show item labels are standard D2 controls we should confirm live (UNVERIFIED).

## D. Open questions the wave-2 engineers should settle live
1. Where are quest bits and the waypoint-active bitmask in 1.14d memory? (Needed for 01 and 03.)
2. Does `d2map` get preset units of rooms that are not yet in view? (Needed for run navigation to exits.)
3. Exact UI geometry: Esc menu, shop grid, inventory/stash/cube cell coordinates, NPC menu rows, skill bar hotkeys at 800x600.
4. Is the sorceress able to attack-in-place with shift-left-click while standing next to the Static Field target etc.
5. Monster resist stats: present in the monster unit's StatList in 1.14d (expected) and are shifted (>>8) for hp.
6. How quickly new drops appear in the unit hash after a kill (affects pickup wait).
7. Does the single-player difficulty dialog appear for every start?

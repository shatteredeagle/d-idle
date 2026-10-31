# Execution: how the bot achieves each goal family

Scope: DOCS ONLY (no live verification). Tags (V) fetched primary/reliable, (S) secondary/snippet, (U) from memory or guessed. "LIVE" = verified live
according to the repo docs. Catalog ids refer to catalog.md. Existing modules (docs/bot.md, nav.md, town.md, combat.md): `nav.Navigator`
(GoTo, GoToUnitOf, ExitTo, TakeWaypoint, ToArea, Route, ActivateWaypoint, Explore), `town.Town` (ToTown, Chores, BackToField, Rebuy, NPC interact + trade
LIVE for Gheed/Akara/Charsi), `combat.Fighter` (Clear, Kill, Buff, Safety), `loot.Looter` (PickUp, Evaluate), `bot.Runner` (planner plan.go, death policy,
watchdog ladder), `launch.Flow` (Cycler: leave/enter game, OOG menus).

Sources: https://classic.battle.net/diablo2exp/quests/act1.shtml .. act5.shtml (V: quest names, NPCs, items, triggers);
https://www.purediablo.com/?p=3457 (S: rewards); docs/archive/prior-art/02-runs.md and 01-leveling.md (kolbot/SoloPlay port notes, S/U); docs/plan/data/runs.json.

## 1. The common step model (what a quest handler is)

A handler is a list of named steps, each a small state machine that can be re-entered after a crash/death (idempotent: "if the item is already in
the pack, skip the fetch"). Steps compose from these primitives; each one already exists unless flagged NEW:

| primitive | built from | status |
|---|---|---|
| `Travel(area)` | `Navigator.ToArea` (waypoint if possible, else `Route` + `ExitTo`), hop verified by area id | exists (bot.runner travel) |
| `Clear(area, until)` / `KillUnique(name)` | `Explore(done = hostile near)` + `Fighter.Clear`/`Kill` + `Looter.PickUp` | exists; boss scripting not done |
| `Talk(npc, option)` | `GoToUnit(npc)` + `input.Interact` + dialog flag 0x08, click option N | NPC open LIVE; option picking NEW (`internal/town` has only trade/stash flows) |
| `FetchObject(preset/obj)` | `GoToUnitOf(TypeObject, ...)` or preset position + `Interact`, verify by item in pack or object mode | partial (waypoint object LIVE); chests/altars NEW |
| `Cube(recipe)` | open cube (stash UI), place items, press Transmute, take result | NEW (`internal/town/cube`) |
| `UseItem(item)` | right-click in pack (tome/scroll/book) | exists for TP scroll (`town.ToTown`); books NEW |
| `Equip/Unequip(item)` | `hand.Transfer` into slot | exists in `town.Rebuy` (UNVERIFIED equip path) |
| `Checkpoint(pred)` | read memory: quest bit / waypoint bit / inventory | NEW reader (`internal/d2/quest.go`, `waypoint.go`), see tracking.md |
| `NewGame(difficulty)` | `launch.Flow` OOG create/join | exists, UNVERIFIED end to end (bot.md) |

Every quest handler ends with `Checkpoint` that confirms completion from the quest bits when the reader exists, else from a proxy (reward item in
pack, NPC dialog text changed, kill event). Without the quest reader the handler is run-once per game session and cannot prove success.

## 2. Goal families: decomposition

### level / reach_level (exists)
`plan.go` picks the area from progression-<diff>.json by clvl, `Travel`, sweep with Buff/Clear/PickUp/Explore, re-plan on level up. Missing: **Leveler**
(spend stat/skill points: HUD buttons show as UI flags 6/7 per bot.md; open character (C) and skill tree (T) panels, click +, verify the stat
changed), respec flow at Akara, difficulty switch. Fail: no point spending means weak character => deaths; the planner's `levelMargin` stays safe.

### quest (NEW goal type)
`Goal{type: quest, params: {quest: "den_of_evil", difficulty}}` => handler (section 3) => done when the quest bit says complete (or proxy). One handler
per catalog `quest.*` row. Handlers call `kill_boss`/`farm_area` internally for the fight part so kills still flow into stats.

### boss run / kill_boss (exists, scripting partial)
Today: boss name -> `runs.json` -> same as farm_area on the final area; the manager counts kills. Missing per boss:
- Locate the boss: preset unit or superunique name (`d2map` preset + monster list), then `Fighter.Kill(unitID)`.
- Special sequences: Diablo (5 seals), Baal (waves), Duriel (tomb + staff), Mephisto (Durance 3, optional moat trick), Nihlathak (corpse explosion),
  Summoner (maze), Ancients (no leaving area), Andariel (poison). See section 4.
- Repeat via `Cycler` (new game) with a "killed this game" set so a boss is not re-looked for in the same game (prior-art 02 common skeleton).

### area / farm_area (exists)
Explore + clear sweep. `area.clear:*` maps to `untilKills` or "cleared" from `ErrExplored`; "visited all" (`area.explore:act1_map`) needs the area-visited set
(NEW, from events/automap, tracking.md).

### wp (NEW)
For each act: iterate the target list; `Route` to the area (walk, not waypoint, until it is active), `ActivateWaypoint` (explores until the preset shows
up, walks, opens panel, closes; exists in nav), verify the bit (reader) else by `TakeWaypoint(area)` listing. Order that minimises walking: follow
the act's natural progression (Act 1: Cold Plains, Stony Field, Dark Wood, Black Marsh, Outer Cloister, Jail 1, Inner Cloister, Catacombs 2 (U, matches runs.json wp ids)).
Pitfall: Act 2 Sewers 2 waypoint, Act 3 Upper Kurast/Travincal/Durance 2 need quest-gated access (orb smash for Durance 3 only, Durance 2 is open, U).

### res (NEW)
- gold: farm profile that favours gold (progression table `gold` purpose) and stash deposit (`Chores` has Gold step). Check = gold + stash gold.
- potions/belt/scrolls: `town.Chores` already shops to belt model (potions-belt.json); the goal just states the target and runs `Chores` with `Force`.
- repair/merc: `Chores` repair exists, merc hire/revive NEW (Act 2 Greiz, Act 1 Kashya, Act 5 Qual-Kehk): `Talk` + trade-like panel.

### item (partial)
`collect_item` counts drops (exists, bot returns an error: "not supported"). Implement: choose the farm area by the item (runes: Countess/Andariel/..
`loot` tiers), run farm_area with `item` filter, count by inventory diff. `item.equip:*` = tier scoring in `loot` + equip click path.

### build (NEW)
Leveler reads `docs/plan/data/build-*.json`: table of (clvl -> skill/stat). On level-up event (or at start) spend unspent points. Verify with the
character sheet read (stats 4/5 are the unspent counters, Player.StatPoints/SkillPoints in d2). Fail: a skill prerequisite not met (click does nothing) =>
read back skill levels and mark the table entry skipped with an error event.

### rune (NEW)
`rune.cube:*` = `Cube(recipe)` with ingredient lists from the pack; `rune.runeword:make` = place base + runes in order in a socketed item (needs a
socketed base: Larzuk socket quest or drops). Recipes and runeword rune lists: docs/plan/data/runewords.json.

### safe (partly exists)
Settings already exist (chicken %, potion %, maxDeaths). New gates: "do not enter NM/Hell below resist X": read resist stats (`Player.Resist` exists).

### diff (NEW)
After `quest.baal` (or Diablo in classic): `Cycler` leaves the game, the OOG layer creates/joins a game with the next difficulty (`launch.Flow`
difficulty dialog, prior-art 08), verify via GetDifficulty byte (LIVE location, only Normal observed: value 0, 1/2 assumed for NM/H, U). Character
resistances drop by the difficulty penalty: planner gate (section 5 of planner.md).

## 3. Quest mechanics 1.14d (what a handler must do)

Format: NPC, route, item, boss, reward, pitfalls. "Req" = required for progression to the next act; "Skip" = may be skipped. Unless tagged, the
route/area ids and flow come from (U) recollection of play experience and kolbot/SoloPlay notes in docs/archive/prior-art/01-02, and must be re-verified live.

### Act 1 (Rogue Encampment, waypoint Cold Plains 3, Stony Field 4, Dark Wood 5, Black Marsh 6)
| quest | steps | reward | req? | pitfalls / handler needs |
|---|---|---|---|---|
| Den of Evil (V npc Akara, kill all monsters incl. Champions and Uniques, V) | Akara talk, walk Blood Moor to cave (area 8), clear every monster, return to Akara | 1 skill point and a respec (S purediablo; the respec is flagged UNVERIFIED in docs/archive/prior-art/01-leveling.md section 3); point goes to the build (repo data docs/plan/data/quest-points.json cites Maxroll) | Skip (but good) | need "den cleared" = quest bit or Akara re-talk. Clear-all means ExitTo(2) does not work from a dungeon (nav.md): leave by walking to the entrance preset and town portal. Normal only for the point in repo data; NM/H: skip (note in quest-points.json) |
| Sisters' Burial Grounds (V Kashya after Den, or entering Burial Grounds, kill Blood Raven, V) | Cold Plains to Burial Grounds (17), kill Blood Raven (ranged, Fire arrows; kite) | Kashya: Rogue mercenary hire (S) | Skip | Kashya turn-in dialog; merc hire panel NEW |
| The Search for Cain (V Akara after Blood Raven, or tree; bark from Tree of Inifuss; touch Cairn Stones in order, V) | Dark Wood: take Scroll from Tree of Inifuss (V item "bark"), to Akara to read, Stony Field cairn stones, portal to Tristram, kill Tristram monsters, free Cain from the cage, talk | free ID from Cain, a ring from Akara (S) | Skip (Cain also needed to talk to for the Act 2 scroll, U) | stone order is hidden/random per game (U): brute force, reading object mode per stone (prior-art 02 Tristram row, kolbot loops); Rakanishu unique in Stony Field; portal window <= 1 min; Tristram is dangerous at clvl < 10 |
| The Forgotten Tower (V trigger: Moldy Tome, V) | Black Marsh (6): Forgotten Tower (20) down to Tower Cellar 5 (25), kill the Countess | chest and rune drops; in MP Free-For-All (V) | Skip | stairs descend, no WP; Countess chest excluded from random chest-opening (prior-art 02) |
| Tools of the Trade (V Charsi, kill the Smith in Barracks, clvl 8, V) | Barracks (28): kill The Smith, pick up Horadric Malus, give to Charsi; choose item to imbue (open item window) | imbue one item (V) | Skip | must hand Malus to Charsi (Talk option NEW) and choose the weapon/armor with the best use; imbue picks a rare with bonuses (S) |
| Sisters to the Slaughter (V Cain or entering Catacombs 4) | Catacombs 4 (37): kill Andariel, return to Rogue Camp, talk to Warriv for the ride | Act 2 access | **Req** | poison (Normal: antidotes, runs.json note); Andariel casts on reaching; avoid melee; after kill the player must TALK to Warriv (option "Go East") NEW |
| Secret Cow Level (purediablo extras, S) | Wirt's Leg (Tristram, cow body, picks up leg), Tome of TP in Cube, transmute in town => portal | cows, King | Skip | needs Cube (Act 2), one kill of the King closes the portal for the difficulty (U) |

### Act 2 (Lut Gholein, WP Lut, Sewers 2 (48), Dry Hills, Halls of Dead 2, Far Oasis, Lost City, Palace Cellar 1, Arcane, Canyon)
| quest | steps | reward | req? | pitfalls |
|---|---|---|---|---|
| Radament's Lair (V Atma, kill creature in the sewers, V; Horadric Scroll starts the Staff, V) | Sewers 1..3 (47-49), kill Radament, pick up Book of Skill (use it), pick up Horadric Scroll chest, talk to Atma, then Cain | 1 skill point (V quest-points.json), potion shop discount (S) | Skip (but the scroll gates the Staff quest, V) | Radament drops a quest item; must use the Book (right-click) NEW; Cain turn-in |
| The Horadric Staff (V Cain; Cube from Halls of the Dead 3; Shaft from Maggot Lair; Amulet from Claw Viper Temple, V; combine in Cube; enter Tal Rasha's tomb, V) | Cube: Halls of the Dead 3 chest (58) after Dry Hills; Amulet: Lost City (44) > Valley of Snakes (45) > Claw Viper 1 > 2, altar; Staff of Kings: Far Oasis (43) > Maggot Lair 1-3, chest; transmute amulet+shaft; go to Canyon of the Magi (46), the right tomb (has the Orifice object); put the Staff in the orifice (needs the staff in pack, click orifice) => portal to Duriel's Lair | Cube kept (S) | **Req** (to reach Duriel) | right tomb is per-game (U); detect by the `HoradricStaffHolder` preset/object (prior-art 02); Cube needs `Cube(recipe)` NEW; Maggot Lair is large; Claw Viper has Viper Amulet guarded by Fangskin |
| The Tainted Sun (V triggered by Lost City, find the Claw Viper temple, V) | part of the Amulet fetch | Viper amulet, sunlight | Req (item) | same as above |
| The Arcane Sanctuary (V Drognan and Jerhyn, V) | Palace Cellar 3 (54) portal after talking to Jerhyn/Drognan (U) | access | Skip | portal is in the palace cellar; talk dialogs NEW |
| The Summoner (V, kill in sanctuary) | WP Arcane (74), Teleport (strongly), find Summoner at the end of one of 4 arms (preset Journal offset (-3,-3), prior-art 02), kill, take red portal to Canyon | portal, tomb location (S) | Skip for progression (U), many speedrun bots skip | maze needs Teleport or a large map exploration time; arm layout random |
| The Seven Tombs (V Jerhyn; after level 20 or other quests, V) | Duriel kill in Duriel's Lair (73); talk to Tyrael, Jerhyn, then Meshif | Act 3 access | **Req** | Duriel freeze/slow, Normal clvl ~24 (runs.json); talk to Meshif and click Sail East (NEW dialog) |

### Act 3 (Kurast Docks, WP Docks, Spider Forest, Great Marsh, Flayer Jungle, Lower Kurast, Bazaar, Upper Kurast, Travincal, Durance 2)
| quest | steps | reward | req? | pitfalls |
|---|---|---|---|---|
| The Golden Bird (V Alkor, a Jade Figurine, V) | Jade Figurine drops from a unique (Sszark? U); give to Meshif then Golden Bird to Alkor (chain U) | +20 life potion (S) | Skip | the exact chain and dropper are U: spike dialog texts |
| Blade of the Old Religion (V Hratli, find Gidbinn, V) | Flayer Jungle (78) Pygmy village, altar, take Gidbinn, give to Hratli | a rare ring (S) | Skip | dangerous Flayer shamans; Hratli turn-in |
| Khalim's Will (V Cain; Heart (Sewers), Brain (Flayer Dungeon), Eye (Spider Cavern), Flail from a High Council member, cube, smash Compelling Orb, V) | Eye: Spider Cavern (85) chest; Heart: Kurast Sewers (92-93) chest; Brain: Flayer Dungeon (88-91) chest; Flail from a Council member at Travincal; cube flail + 3 relics; equip Khalim's Will; attack Compelling Orb | opens Mephisto's prison (V) | **Req** | quest items cannot be sold/dropped; equip then re-equip weapon; Council fight is Hell; orb smash needs weapon equipped, then switch back |
| Lam Esen's Tome (V Alkor, Ruined Temple, V) | Kurast Bazaar (80) > Ruined Temple (94) altar; take tome; talk to Alkor | +5 stat points (S) | Skip | portal of the temple is behind enemies; `use` tome |
| The Blackened Temple (V Ormus, kill High Council, V) | Travincal (83): all 3 Council members | no reward (S) | Req (orb) | Council casts Lightning/Cold/Fire; split fighting; Mana Burn on Hell |
| The Guardian (V Ormus, reach Mephisto, V) | Durance 3 (102), kill Mephisto, take red portal to the Fortress | Act 4 | **Req** | Durance 2 WP; Mephisto summons; the moat trick is optional |

### Act 4 (Pandemonium Fortress, WP Fortress, City of the Damned, River of Flame)
| quest | steps | reward | req? | pitfalls |
|---|---|---|---|---|
| The Fallen Angel (V Tyrael, find Izual, Plains of Despair, V) | Outer Steppes 104 > Plains of Despair 105 (no waypoint, walk from the Fortress, U); kill Izual; talk Tyrael | 2 skill points (V) | Skip (but worth it) | Izual freezes; Act 4 monsters are dangerous at < clvl 24 |
| Hellforge (V Cain, Mephisto's Soulstone + Hammer, V) | Mephisto's Soulstone: pick up the quest item dropped by Mephisto; River of Flame (107) forge (kill Hephasto the Armorer to get the Hammer); smash soulstone | gems, a rune in expansion (S) | Skip | the Soulstone is a ground quest item: **the bot must not skip loot tier for quest items** (loot has a `quest` field) |
| Terror's End (V Tyrael after soulstone, kill Diablo, V) | Chaos Sanctuary (108): open 5 seals (each seal object click), kill Vizier, de Seis, Infector, then Diablo; talk to Tyrael; take portal to Harrogath | Act 5 | **Req** | seal order and spawn timing (prior-art 02 Diablo row); walls of fire/lightning; dangerous in Hell; Tyrael portal NEW |

### Act 5 (Harrogath, WP Harrogath, Frigid Highlands 111, Arreat Plateau 112, Crystalline Passage 113, Halls of Pain 123, Glacial Trail 115, Frozen Tundra 117, Ancients' Way 118, WK2 129)
| quest | steps | reward | req? | pitfalls |
|---|---|---|---|---|
| Siege on Harrogath (V Larzuk or Shenk, V) | Bloody Foothills (110): kill Shenk the Overseer (V SuperUnique in a special area); speak to Larzuk | Larzuk adds sockets (S) | **Req**? (gives Rescue, U) | choose socket target; sockets are stackable only 1 time per item |
| Rescue on Mount Arreat (V Qual-Kehk) | Frigid Highlands: break 3 barbarian doors, return; can FAIL (V "create a new game") | 3 runes (Tal, Ral, Ort, S), barbarian mercs | Skip | quest can fail; detect failure message; Cycler new game |
| Prison of Ice (V Malah or approach Anya in Frozen River, V) | Frigid Highlands > Frozen River (114): find Anya, open ice, return to Harrogath, talk to Malah and Anya | +10 all resists via Scroll (S) | **Req** (portal to Nihlathak, U) | read the scroll; need Anya portal (Anya talk option NEW) |
| Betrayal of Harrogath (V Anya, kill Nihlathak, Halls of Vaught, V) | Anya red portal or Halls of Pain; Halls of Vaught (124) kill Nihlathak (Corpse Explosion, V) | Anya personalises an item (S) | **Req** | corpse explosion: kill with ranged, ignore or teleport; Hell Nihlathak has Mana Burn |
| Rite of Passage (V Qual-Kehk, Ancients, clvl 20/40/60 and stay in area, V) | Arreat Summit (120): kill Madawc, Talic, Korlic without leaving, no TP | XP bonus (S) | **Req** | leaving or town portal resets them (V); handler must disable town portal for the fight (`Fighter.Safety` TownPortal action must be suppressed) |
| Eve of Destruction (V the Ancients) | WK2 (129) to WK3 (130) to Throne (131): clear 5 waves, enter chamber (132), kill Baal | finish difficulty | **Req** | Baal's fakes (V); waves (prior-art 02); town portal is allowed here (only the Ancients forbid it) |

## 4. Boss specifics the Fighter needs (NEW `internal/bot/runs/*.go`)
Per boss: approach point (preset/coordinate from prior-art 02), danger affixes (docs/plan/data/danger-table.json), resist check, buff, kill, loot,
exit. First targets by value: Countess (runes), Andariel, Mephisto, Pindleskin, Diablo, Baal. Run skeleton and "killed this game" set: prior-art 02.

## 5. Failure modes and recovery
| failure | detect | recovery |
|---|---|---|
| quest item lost / not found | pack scan for item class, fetch step re-entered | re-clear area; max 3 tries then fail with note |
| NPC dialog option not found | UI flag 0x08 never appears, or option text not matching | close dialog (Esc, careful: Esc with no panel opens menu, mem.md), re-approach, fail goal after 3 tries |
| wrong tomb / wrong stone order | object mode change absent, or no portal after N s | brute force next candidate; reload game via Cycler |
| boss reset (Ancients) | monsters regain life / portal opened | restart from Arreat Summit entrance; TP suppressed |
| death in quest area | death policy (bot.md) | spiral breaker avoids area; goal pauses after `pauseDeathsPerHour`; planner goes level elsewhere (`level.reach`) |
| game restart mid-quest (state lost) | quest bits differ from handler step | resume by reading bits and inventory, never from handler memory |
| quest can fail (Rescue, V) | quest log text / bit | accept, skip, record in goal note |
| unknown NPC position | town tables per act (prior-art 05) differ per character (docs/bot.md: this town was at different coordinates) | locate by unit scan (`GoToUnitOf`), not by constants |
| ran out of gold/potions | `town.NeedsTrip` | resupply trip exists |
| difficulty penalty too large | resist read before NM/H (planner) | farm resistance gear / stay |

## 6. New bot capabilities (concrete)
1. `internal/d2`: `Quests(diff)` and `Waypoints(diff)` readers from PlayerData (UNVERIFIED offsets, see tracking.md spike list). Package `d2/quest.go`.
2. `internal/town`: dialog layer: `Talk(npc, optionIndex)` and quest reward/option clicking, hire merc, Larzuk/Charsi/Anya/Tyrael interactions. Today only open + trade LIVE.
3. `internal/bot/runs`: boss scripts and quest handlers (Tristram stones, Cube staff, Duriel tomb detection, Durance orb, Hellforge, Chaos seals, Ancients/Baal waves).
4. Cube and socketing: `internal/town/cube` (Transmute button, place items) and runeword socketing; needed for Staff, Khalim's Will, runes, Wirt's leg.
5. Leveler + difficulty switch + act travel: `internal/bot/level.go` (stat/skill/respec), `launch.Flow` difficulty dialog, Warriv/Meshif/Tyrael act changes (config has only `acts: [1]`).
6. Quest-item aware loot: loot already has a `quest` match field; the pickit must always take quest items (Malus, Soulstone, relics, Jade Figurine).
7. Mercenary management: hire, revive, and equip (not present).

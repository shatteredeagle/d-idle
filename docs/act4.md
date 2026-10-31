# Normal Act 4 plan (agent ACT3PLAN, docs only)

> status: OFFLINE PLAN: never run live; 400+ code commits since this was last touched (doc audit 2026-10-30). Treat every row as U. Current state: docs/plan/STATUS.md.

Companion of `docs/act3.md` (shared work list X1..X7 lives there) and `docs/act5.md`. Target classic LoD 1.14d. Docs only, no game, no Go.

Tags: **V** verified (our code at f5b7197, other docs' live results cited, REAL 1.14d tables on the worker host read 2026-10-29, or a page I fetched), **S** secondary source I fetched (URL),
**U** unverified / memory. Sources: Arreat Summit `https://classic.battle.net/diablo2exp/quests/act4.shtml` and `.../rewards.shtml` (V text);
Kolbot `https://raw.githubusercontent.com/blizzhackers/kolbot/master/d2bs/kolbot/libs/bots/{Izual,Hephasto,Diablo,Questing}.js`,
`.../libs/common/Common.js` (Diablo layout, paths; S), `.../tools/RushThread.js`, `.../libs/modules/sdk.js`; repo copies of the Diablo node paths in `docs/archive/prior-art/02-runs.md` (S, KB port notes).

## 1 What exists today (V, code at f5b7197)

| layer | Act 4 | note |
|---|---|---|
| `saves` words | `act4_intro` 24, `izual` 25, `hellforge` 26, `terrors_end` 27, `act4_done` 28 (`internal/saves/progress.go`) | 37 and 40 (Act 5) are V live; Act 4 words S (docs/saves.md: the Act V log tab shows only once word 27 is done, V live) |
| `quest.Hops` | `A3->A4 red portal` (from area 102, `ObjRedPortalA3 = 342`, S, per-act since ACT45BOT) and `A4->A5 Tyrael` (class 367, row "Travel to Harrogath", requires `terrors_end`, U) | tested with fakes only; never run by the bot |
| `quest.Menus` | Tyrael 367 main + talk rows (U) | no Jamella, Halbu, Cain A4 |
| `task` | nothing (no Act 4 ids, nodes, campaign) | |
| `boss` | no Izual, Hephasto, Diablo script; the runner does not yet know seals | framework: `RunWithEntry(enter)` (Duriel uses it) is the natural slot for a seal executor |
| `nav` | `wpLevels[3] = {103,106,107}` (matches the table, V); tab x for act 4 = 287 extrapolated (U); no hints, no `knownWaypointPos` | |
| `town` | `Acts[3]`: town 103, Heal/Shop/Gamble/Keys Jamella 405, Repair Halbu 257, waypoint object 398, no merc | classes V (monstats), roles U |

## 2 Quest chain (text V Arreat act4.shtml)

| quest (word) | trigger | work | reward | tags |
|---|---|---|---|---|
| Act 4 intro (24) | arrival | none | | S word |
| The Fallen Angel (25) | Tyrael gives it | City of the Damned waypoint (106) -> Plains of Despair (105), kill Izual (class 256), talk to Tyrael in the Fortress | **2 skill points** (V rewards.shtml) | Izual preset anchor S (Kolbot Izual.js: `moveToPreset(PlainsofDespair, Monster, Izual)`); the Plains have no waypoint (V table) |
| The Hellforge (26) | "Cain gives it as soon as you enter Act IV" | needs `mss` Mephisto's Soulstone (picked up in Act 3, Durance 3) and the Hell Forge Hammer `hfh` (2x3, V items) from Hephasto the Armorer (SU 41, class 409, standing at the Hellforge object 376 in River of Flame 107, S Kolbot Hephasto.js); put the Soulstone on the forge, hit it with the Hammer | 4 gems; Expansion adds one rune per difficulty: Normal El..Amn (V rewards.shtml) | placement procedure U (equip hammer vs click item; unknown which); optional, `Skip: true` |
| Terror's End (27) | "Tyrael after destroying Mephisto's Soulstone" (V text), but Kolbot's `Questing.diablo` runs without it (S) | Chaos Sanctuary (108): open the five seals, kill the three seal bosses, Diablo (class 243); Tyrael appears and a portal to Harrogath opens | Act 5 access | **Req** |
| Act 4 done (28) | Tyrael's talk / portal to Harrogath | `A4->A5` hop; needs the Expansion flag (V-02 in docs/progression/verification.md) | | S word |

Quest facts the planner must know: Izual and Hellforge are independent of Diablo; Hellforge is the only quest that needs an item from the previous act (a Mephisto
kill without picking the Soulstone up forces a new game and another kill, S/U).

## 3 Areas and waypoints (V levels.txt)

| id | data name (in-game) | wp (number) | MonLvl N/NM/H | Vis |
|---|---|---|---|---|
| 103 | The Pandemonium Fortress | 27 (object 398 V) | town | none |
| 104 | Outer Steppes | none | 26/56/82 | none |
| 105 | Plains of Despair (Izual) | none | 26/56/83 | none |
| 106 | City of the Damned | 28 | 27/57/84 | 107 |
| 107 | River of Flame (Hellforge) | 29 | 27/57/85 | 106, 108 |
| 108 | Chaos Sanctum (Chaos Sanctuary) | none | 28/58/85 | 107 |

The links 103-104-105-106 are level crossings, not in Vis (V): `act4Hints = {103,104},{104,105},{105,106}` (S order from the map, U edges). River of Flame -> Chaos Sanctuary
is a Vis link (V). The way into Act 4 and back: the red portal object from Durance 3 lands in the Fortress (V live: `act Multi 4` arrives in 103, docs/progression/results.md V-03).
Waypoint panel: tab IV at x 287 is an extrapolation (U), only three rows; Act 4 does not need the icon-column rule (rows < 7, U).

## 4 NPCs, objects, items

NPCs (V monstats): Tyrael `tyrael2` 367, Jamella 405 (also `town.Acts` Heal/Shop/Gamble), Halbu 257 (Repair), Cain `cain4` 246 or `cain5` 265 (which stands in Act 4 is U), Izual spirit `izualghost` 406 (appears after the kill, S). No mercenary in Act 4 (`town.Acts[3].Merc = nil`; hireling table has Act 1, 2, 3, 5 sellers, V for 3 and 5).
Objects (V objects.txt, roles S Kolbot sdk.js): 376 Hellforge, seals 392, 393 (Infector), 394 (Seis), 395, 396 (Vizier) all named "Seal", 131 "Dummy" (Kolbot `SealGlow`), 255 "Dummy" (`DiabloStar`), 398 Act 4 town waypoint, 324 another waypoint (field), 566 named **"Harrogath"** (V name: the portal to Act 5, Kolbot `RedPortalToAct5`).
Items (V): `mss` Mephisto's Soulstone 1x1, `hfh` Hell Forge Hammer 2x3, `luv` The Black the NAS Key (unclear role, U).

## 5 Cube

None needed in Act 4 (the Hellforge is an object interaction, not a recipe, V: no cubemain row mentions `mss`/`hfh`).

## 6 Bosses and fights

| boss | where | prerequisite | V table data | fight facts (S Kolbot unless noted) |
|---|---|---|---|---|
| Izual | Plains of Despair 105 (walk from City 106) | Fortress reached; floor clvl 30 (progression-normal, R) | class 256, level 29/60/86; **all resists 30, cold 75** at every difficulty; Normal HP in the table is the raw column (not scaled) | preset monster anchor, stand 50-60 subtiles off first (RushThread `moveIntoPos`), then kill; he freezes (U, older docs) so do not melee unprotected; cold builds are weak against 75 percent |
| Hephasto the Armorer | River of Flame 107 at the Hellforge (object 376 preset) | Soulstone in the pack | SU 41 (V), class 409, level 25/60/88; Normal poison -50 (V), Hell fire 75 phys 50 | kill by name at the forge; skip if no Soulstone |
| Seal bosses | Chaos Sanctuary 108 | none | Grand Vizier of Chaos SU 38 (`fingermage3`), Lord De Seis SU 37 (`doomknight3`), Infector of Souls SU 36 (`megademon3`) (V names/classes); each opens with a defender pack (S: Oblivion Knights, Venom Lords, Storm Casters) | see 6.1 |
| Diablo | star point (7791,5293) | all five seals opened | class 243, level 40/62/94, resists Normal 33 fire/light/cold, 50 poison; NM 50; Hell phys 45, elements 50 (V table); TC `Diabloq` quest TC | spawns after the last seal opens: Kolbot waits up to 30 s at the star (packet 0x89 0x0C "lights"), a 250 ms poll of the monster list is our substitute (U); Fire Storm/lightning hose danger objects (S, danger kind "fire wall" in docs/progression/bosses.md); kill, loot, Tyrael portal (object 566) |

### 6.1 The seal executor (the one genuinely new mechanism)

Geometry is fixed (S, `Common.js` `Common.Diablo` and docs/archive/prior-art/02-runs.md): entrance (7790,5544), star (7791,5293). Each of three seal groups has layout A or B, detected from
the preset position of one seal object: Vizier seal 396 with preset y == 5275, Seis seal 394 with x == 7773, Infector seal 392 with x == 7893 (the match gives layout 1 = "A"). Node paths:
entrance->star `7794,5517 7791,5491 7768,5459 7775,5424 7817,5458 7777,5408 7769,5379 7777,5357 7809,5359 7805,5330 7780,5317 7791,5293`; star->Vizier A `7759,5295 7734,5295 7716,5295 7718,5276 7697,5292 7678,5293 7665,5276 7662,5314`, B `7759,5295 7734,5295 7716,5295 7701,5315 7666,5313 7653,5284`;
star->Seis A `7781,5259 7805,5258 7802,5237 7776,5228 7775,5205 7804,5193 7814,5169 7788,5153`, B `... 7776,5228 7811,5218 7807,5194 7779,5193 7774,5160 7803,5154`; star->Infector A `7809,5268 7834,5306 7852,5280 7852,5310 7869,5294 7895,5295 7919,5290`, B `... 7869,5294 7895,5274 7927,5275 7932,5297 7923,5313`.
Seal open = object mode != 0 (S); up to 5 tries; the Infector has two seals (392 and 393), the Vizier two (395, 396); five seal objects in all. Order Vizier, Seis, Infector (S). The bot must walk or teleport node to node with `combat.Clear` radius 30, which is what `nav` teleport hops and `Fighter.Clear` already do individually (V live in Act 2 for teleport hops, Clear in all acts).
Unknowns: the object-mode value after opening and the seal glow object (131) behaviour (U), and whether `Presets` lists the seal objects before the room streams in (it did not for the Act 2 chests, V docs/act2.md R1/F1; Kolbot reads presets from the map, so our `d2map` preset list is the first thing to check live).

Dangers per class (U except the V columns): Sorceress - ranged and teleporting through the sanctuary is the standard answer, but Normal Act 4 packs (level 28) kill an under-levelled caster quickly (Act 2 live finding X7: clvl 24 died in seconds without gear, V); Paladin/Barbarian - melee in the seal rooms draws the whole defender pack, retreat points matter (`boss.Hazard CrowdedAndHurt`, exists V); cold-only builds are poor against Izual (cold 75, V); poison builds do half damage against Diablo (50, V); physical damage is unresisted by Diablo in Normal and 30 percent by Izual (V). Hell: Diablo physical 45 and elements 50 (V).
Level floors (R, runs.json/progression-normal, source of their numbers U): Izual 30, Diablo 30/55/80.

## 7 Support matrix and gaps (V code)

| capability | have | gap |
|---|---|---|
| Hop A3->A4 | step data, `ObjRedPortal = 60` U | object id (342 per Kolbot, S, vs 60), portal only exists in the game of the Mephisto kill (S) |
| Act 4 constants and NPC rows | none | `quest/act4.go`, menus for Tyrael/Cain/Jamella/Halbu (U rows) |
| Izual leg | boss framework, `Anchor.Preset{Kind monster}` supports it (V: Blood Raven uses `PresetNPC` 805) | script with the preset id (monster id from `Presets`, U), `Teleport`-free walk from City 106 |
| Hellforge | none | item placement and hammer swing (new capability, see X5), Hephasto script, Soulstone kept in the pack across the act hop (it is a quest item, V `Quest`) |
| Diablo | none | seal executor (6.1), `Script` for Diablo, hazard for fire-wall objects (kind exists, producer U), `HasTP` handling is fine |
| Chaos Sanctuary navigation | `nav.ToArea`, teleport hops | `act4Hints`, a node follower that clears per node |
| Hop A4->A5 | `quest.Hops[3]` | Tyrael menu row, or portal object 566; verify the class that stands in the Fortress after Diablo |

## 8 Work list (A4-n; shared X1..X7 in docs/act3.md)

- **A4-1 quest data (d2-worker)**: `internal/quest/act4.go` and `act4_menus.go`: area ids (V table), NPC classes V, objects (376, 392-396, 398, 566), items (`mss`, `hfh`), `Act4Order`; `TestAct4ConstantsAgainstRealTables` (skips without `/tmp/d2mpq`).
- **A4-2 task nodes (d2-worker)**: `internal/task/act4.go`: `izual` (travel City wp, walk to 105, kill, Tyrael talk), `hellforge` (optional, ready when `Items["Mephisto's Soulstone"] > 0` and `Hell Forge Hammer` obtainable), `diablo`, `act4_hop`; campaign `NormalAct4` and `NormalActs1234`; floors 30; reader ids for words 25-28.
- **A4-3 boss scripts (d2-worker)**: `internal/boss/act4.go`: `Izual` (Match class 256, preset anchor, `Kite` off, cold note in hazards), `Hephasto` (Match name, preset object 376), `Diablo` (Match class 243, Anchor Boss at the star, `RunWithEntry` with the seal executor as `enter`). Fake-world tests.
- **A4-4 seal executor (d2-worker, the big one)**: `internal/boss/seals.go`: pure data (layouts A/B, node paths from 6.1) plus a runner over a small interface (`Presets`, `GoTo`, `ClearAround`, `UseObject`, monster poll); layout detection from preset coordinates with tests for both layouts per seal; "seal open" and "boss spawned" detection with timeouts; Diablo spawn wait 30 s. 100 percent offline (fake world with the preset coordinates from the layouts).
- **A4-5 nav hints and Chaos walk (d2-worker, then d2-live)**: `act4Hints`, `knownWaypointPos[103]` after a live reading; live: first ExitTo 103->104->105->106, River of Flame -> 108 arrival point, presets of the five seals before streaming.
- **A4-6 Hellforge mechanics (d2-live after X5)**: find out how the Soulstone is placed and the Hammer used (equip + attack the forge object vs cursor drop), record the quest word change (26), rewards; code afterwards.
- **A4-7 hop A3->A4 and A4->A5 (d2-live)**: check object 342/60 in Durance 3 after a Mephisto kill, the Fortress arrival, and Tyrael vs object 566 after Diablo; flip the `Tag`s in `hops.go`.
- **A4-8 live pass for Izual and Diablo (d2-live)**: cheat-load a clvl 30+ character with gear (the `d2cheat act N 4`, waypoints all), run `Izual`, then the seal executor per seal, then Diablo; per Act 2 experience a character without gear dies at once, so decide the gear/merc source first (Act 3 Asheara merc, docs/act3.md 4).

## 9 Open questions

1. Hellforge placement procedure and whether the rune reward needs the Expansion flag per character (V text says "Expansion includes ... one rune per difficulty").
2. Whether `Presets` exposes the seal objects (392-396) and the Izual monster preset before the area streams in (our Act 2 chests did not, V).
3. Object-mode value of an opened seal and the exact trigger that spawns Diablo (all seals vs. all three bosses dead): Kolbot waits after the last seal, U for 1.14d.
4. Which Cain class stands in the Fortress and whether he is needed (Hellforge is given by Cain per Arreat, V text).
5. Whether `levels.txt` Teleport column 1 hides an actual no-teleport zone in the Chaos Sanctuary (the column is 1 everywhere in Acts 3-5 and wrong for towns, V, so it proves nothing; Kolbot's Chaos routes teleport, S).

# Normal Act 3 plan (agent ACT3PLAN, docs only) - also holds the shared work list for Acts 3-5

> status: OFFLINE PLAN: never run live; 250+ code commits since this was last touched (doc audit 2026-10-30). Treat every row as U. Current state: docs/plan/STATUS.md.

Plan for the campaign bot, same style as `docs/act2.md`. Nothing here ran against a live game; no Go was changed. Companion files:
`docs/act4.md`, `docs/act5.md`. Target: classic LoD 1.14d (Normal first; Nightmare/Hell notes only where the tables show a hard gate).

Tags (as in AGENTS.md): **V** = verified, either from our code (`grep` on this checkout, `f5b7197`), live results in other docs (cited), or read from the REAL
1.14d tables on the worker host on 2026-10-29 (`data.Open("/tmp/d2mpq")`, throwaway test, not committed: levels, monstats, superuniques, objects, items, cubemain,
hireling), or a source page I fetched (URL given). **S** = secondary source I fetched (Kolbot source on GitHub, URL given). **U** = unverified / from memory.
D2R-only facts: none used; Kolbot is a classic-compatible bot but its constants are for 1.14d-era D2BS, so its coordinates stay S.

Sources fetched: Arreat Summit quest pages `https://classic.battle.net/diablo2exp/quests/act3.shtml` (+ act4, act5, `rewards.shtml`), V text of the quests;
Kolbot `https://raw.githubusercontent.com/blizzhackers/kolbot/master/d2bs/kolbot/libs/bots/{Mephisto,Travincal,Izual,Hephasto,Nihlathak,Questing,Rusher}.js`,
`.../d2bs/kolbot/tools/RushThread.js`, `.../libs/modules/sdk.js` (constants), S.

## 1 What exists today (V, code at f5b7197)

| layer | Act 3 | note |
|---|---|---|
| `quest/saves` | words 16..23 in `saves.questTable` (`act3_intro` 16, `lam_esen` 17, `khalims_will` 18, `blade_old_religion` 19, `golden_bird` 20, `blackened_temple` 21, `guardian` 22, `act3_done` 23) | word 20 = Golden Bird V live (docs/progression/quests.md); the others S order, U bits |
| `quest.Hops` | `A2->A3 Meshif` (Pre `seven_tombs`, ends in Kurast Docks 75) and `A3->A4 red portal` (FromArea 102, `ObjRedPortalA3 = 342`, S, per-act since ACT45BOT) | `RunHop` is tested with fakes only; no bot adapter runs hops (nothing in `cmd/d2bot` calls `quest.Hops`) |
| `quest.Menus` | Meshif A3 (264: Talk / Sail West / Cancel, U) in `act2_menus.go`; Hratli 253 (const only) | no rows for Hratli, Alkor, Ormus, Asheara, Cain A3 |
| `task` | `KurastDocks Area = 75` only; no Act 3 quest ids, nodes or campaign | `NormalAct1` and `NormalAct2` exist; **only `normal_act1` is reachable** (`goals.CampaignNormalAct1`, `internal/bot/runner.go:471`) |
| `boss` | nothing for Act 3 (`All()` = Acts 1+2) and `cmd/d2bot/boss.go:276` builds the runner from `boss.Act1()` only, so even Act 2 scripts are not reachable | framework has `NoTP`, `Teleport` anchor, `Lure`, `Kite`, `Stand`, hazards |
| `nav` | `wpLevels[2] = {75,76,77,78,79,80,81,83,101}` (matches the table, below); tab x for act 3 = 230 V live (docs/act2.md W4); no `knownWaypointPos` or hints for Act 3 | outdoor links are not in levels.txt Vis (see 3) |
| `town` | `Acts[2]`: town 75, Heal/Shop Ormus 255, Repair Hratli 253, Gamble Alkor 254, Merc Asheara 252, waypoint object 237 | NPC classes V (monstats); roles U |
| `cube` | the flow is V live (docs/act2.md C3, `tools/rigs/act2live cubemix`); `town.Transmute` is fake-tested only (ACT2CODE) | the Khalim recipe has 4 inputs, never run |

## 2 Quest chain (words per `saves`; text V Arreat act3.shtml)

| quest (word) | trigger / giver (V Arreat) | work | reward | tags |
|---|---|---|---|---|
| Act 3 intro (16) | arrival, Hratli spoke | talk Hratli | none | S word |
| The Golden Bird (20) | "given by finding a Jade Figurine": a unique monster carries `j34` | pick up `j34`, give to Meshif (A3, 264) who returns `g34`, give to Alkor (254) | Alkor makes a potion: +20 life permanently (V rewards.shtml) | items V (misc.txt `j34` 1x2, `g34` 1x2, `xyz`); dropper U (candidates in the super unique table: Sszark the Burning SU 21 `arach4`, Witch Doctor Endugu SU 22 `fetishshaman4`); Meshif step U |
| Blade of the Old Religion (19) | Hratli after the Golden Bird "or by entering the Flayer Jungle" | Flayer Jungle (78) Pygmy village, altar object 251 "gidbinn altar" / 252 "gidbinn" (V names; Kolbot sdk calls 252 `GidbinnAltar`, S), items `d33` Decoy Gidbinn, `g33` The Gidbinn (V), deliver to Hratli | rare ring (S purediablo, not on Arreat rewards page) | dropper/flow U; skippable |
| Lam Esen's Tome (17) | Alkor "after Blade ... and you must have entered the Kurast Bazaar" (V text) | Kurast Bazaar (80) -> Ruined Temple (94), altar object 193 (V "LamTome"), take `bbb` (V 2x2), give to Alkor | +5 stat points (V rewards.shtml) | Kolbot rushes it without the Blade (S, RushThread `lamesen`); whether the altar needs the quest started U |
| Khalim's Will (18) | Cain "after the Golden Bird or by entering the Great Marsh" | relics: Eye `qey` in Spider Cavern (85), Heart `qhr` in Kurast Sewers 2 (93), Brain `qbr` in Flayer Dungeon 3 (91) (V text tip + level names; chest classes 405/406/407, Kolbot assigns Heart/Brain/Eye in that order, S; all three exist V as "chest"); Flail `qf1` drops from the High Council (below); cube `qf1+qhr+qey+qbr -> qf2` (V cubemain row "Super Khalim Flail", enabled); equip Khalim's Will, left-click the Compelling Orb (object 404 V) in Travincal | opens the stairs to Durance of Hate (S Kolbot `DuranceEntryStairs` 386, V exists as "Dummy") | **Req**; Flail dropper is "one of the Council" (V text); which member U (all six council super uniques share TC "Council", V) |
| The Blackened Temple (21) | Ormus "after Lam Esen or by entering the Kurast Causeway" | kill the three High Council members in Travincal (83) | none | skippable by Ormus text but the Flail needs the kill anyway |
| The Guardian (22) | Ormus "after smashing the Compelling Orb" | Durance of Hate 1 -> 2 (waypoint) -> 3, kill Mephisto (class 242), take the red portal | Act 4 | **Req** |
| Act 3 done (23) | red portal / arrival in Fortress | `A3->A4` hop | | S word; the cheat `act N` (V-03) lands in 103 |

Critical path for the bot: Khalim's Will chain, Council kill, orb, Mephisto. Optional but cheap: Golden Bird (+20 life), Lam Esen (+5 stat points), Blade.
The existing `catalog.md`/`execution.md` rows agree with this (they cite the same Arreat pages).

## 3 Areas and waypoints (V, levels.txt 1.14d; names are the data names, in-game names in brackets where they differ)

Waypoint column = waypoint number (index into the 39-bit mask; 255 = none). `MonLvl` = Normal/Nightmare/Hell. Teleport column read 1 for every Act 3-5 level
including the towns, but Teleport does NOT work in town (V live, docs/act2.md X3), so the column is not a usable capability flag outside Duriel's Lair (2).

| id | name | wp | MonLvl N/NM/H | Vis (listed neighbours) |
|---|---|---|---|---|
| 75 | Kurast Docktown | 18 (object 237) | town | none |
| 76 | Spider Forest | 19 | 21/49/79 | 84 Spider Cave, 85 Spider Cavern |
| 77 | Great Marsh | 20 | 21/50/80 | none |
| 78 | Flayer Jungle | 21 | 22/50/80 | 86 Swampy Pit 1, 88 Flayer Dungeon 1 |
| 79 | Lower Kurast | 22 | 22/52/80 | none |
| 80 | Kurast Bazaar | 23 | 22/52/81 | 92 Sewers 1, 94 Ruined Temple, 95 Disused Fane |
| 81 | Upper Kurast | 24 | 23/52/81 | 92 Sewers 1, 96 Forgotten Reliquary, 97 Forgotten Temple |
| 82 | Kurast Causeway | none | 24/53/81 | 98 Ruined Fane, 99 Disused Reliquary |
| 83 | Travincal | 25 | 24/54/82 | 100 Durance of Hate 1 |
| 84, 85 | Spider Cave, Spider Cavern (Eye chest) | none | 21/50/79 | back to 76 |
| 86, 87, 90 | Swampy Pit 1, 2, 3 | none | 21/51/80..82 | chain |
| 88, 89, 91 | Flayer Dungeon 1, 2, 3 (Brain chest) | none | 22/51/81..83 | chain |
| 92, 93 | Kurast Sewers 1, 2 (Heart chest in 93) | none | 23/52/84, 24/53/85 | 92 from 80 and 81, 93 from 92 |
| 94 | Ruined Temple (Lam Esen altar) | none | 23/53/84 | 80 |
| 100, 101, 102 | Durance of Hate 1, 2 (wp 26), 3 (Mephisto) | 255, 26, 255 | 25/55/83 | 100 <-> 83, 101; 101 <-> 100, 102 |

Outdoor links 75-76-77-78-79-80-81-82-83 are level-edge crossings, not in Vis (same finding as Act 2, docs/act2.md nav). Which edge joins which neighbour is U
except the order by waypoint list (Kurast Docks -> Spider Forest -> Great Marsh -> Flayer Jungle -> Lower Kurast -> Bazaar -> Upper Kurast -> Causeway -> Travincal), U;
the Docks also touch Lower Kurast in my memory (U). nav needs `act3Hints` like `act2Hints`, confirmed by the first ExitTo (docs/act2.md "nav").
Waypoint panel: act 3 tab x = 230 (V live), row order in `wpLevels[2]` equals the table's waypoint numbers 18..26 (V). Rows 7 and 8 (Travincal, Durance 2) probably need the icon column click like Act 2 rows 7/8 (U, `wpPanel.IconFrom = 7` already applies to every act).
Gate: Durance of Hate 100 is reachable only after the orb is destroyed (S Kolbot rush: waypoint 83 -> orb -> stairs); Durance 2 waypoint then opens the shortcut (`Pather.useWaypoint(DuranceofHateLvl2)` in Mephisto.js, S).

## 4 NPCs and objects

NPC classes (V monstats): Hratli 253, Alkor 254, Ormus 255, Asheara 252, Meshif (A3) 264, Cain `cain3` 245 (class that stands in Act 3 is U, `town.Acts` lists all six), Tyrael `tyrael2` 367 is Act 4. Positions are per-seed, never constants (V live in Act 2: unit scan, `townlive units`).
Objects (V objects.txt names; role S Kolbot sdk.js): 193 LamTome altar, 251/252 gidbinn altar/gidbinn, 404 compellingorb, 405/406/407 chest (relic chests), 366 sewer stairs, 367 sewer lever, 386 Durance entry stairs, 237 Act 3 waypoint, 342 "portal" (Kolbot `RedPortalToAct4`, S; our `ObjRedPortal = 60` is U and probably wrong: 60 is the generic "Portal" PP seen at the Arcane/Canyon portal, V live docs/act2.md objs). The Kurast Sewers lever/stairs (366/367) are the entrance trick for the Heart level (S Kolbot constants; behaviour U).
Items (V): `j34` Jade Figurine, `g34` The Golden Bird, `d33`/`g33` Decoy/real Gidbinn, `bbb` Lam Esen's Tome, `qey` `qhr` `qbr` Khalim's Eye/Heart/Brain (1x1), `qf1` Khalim's Flail (2x3), `qf2` Khalim's Will (2x3), `luv` "The Black the NAS Key" (1x2, Act 3/4 item, role U), `mss` Mephisto's Soulstone (1x1). All carry `Quest = true`; ground clicks auto-equip fitting quest items (V finding P1, docs/act2.md), so `loot.unwear` must list the new codes (it lists msf, vip, hst, box, tr1, ass only).
Mercenary: Asheara 252 hires the Eastern Sorceror (Iron Wolf): Fire (Inferno, Fire Ball), Cold (Glacial Spike, Frozen Armor, Ice Blast), Lightning (Charged Bolt, Lightning) rows; Normal rows from character level 15 (V hireling.txt). A cold merc helps against physical-only characters little but slows; U which row to pick (Act 2 Defiance is the existing default).

## 5 Cube recipes needed (V cubemain)

| output | inputs | note |
|---|---|---|
| `qf2` Khalim's Will | `qf1 + qhr + qey + qbr` | 2x3 + three 1x1 = 9 of 12 cube cells; `PlanCubeLayout` should place it (first fit, tallest first) - add a test with these sizes |
| `hst` | `msf + vip` | already V live |

## 6 Bosses and fights

| boss | where | prerequisite | V data (table) | fight facts |
|---|---|---|---|---|
| High Council (3) | Travincal 83, waypoint 25 | none for the kill; the orb needs the Flail | classes 345/346/347 (V), super uniques 26..31 (V: Ismail Vilehand, Geleb Flamefinger, Bremm Sparkfist, Toorc Icefist, Wyand Voidbringer, Maffer Dragonhand; `MinGrp 2`/0 for the six, which three spawn is U); level 25/56/88; Normal resists all 0; Hell: member 1 fire 120, members 2/3 lightning 100 (immune) | Kolbot stands at waypoint + (101,-56) and clears all three (S Travincal.js); Leap/Teleport variants use offsets (+60,-53) etc. They cast lightning/fire/cold and Mana Burn is a Hell risk (danger-table, S). Flail drop: `Council` TC |
| Mephisto | Durance 3 (102) via waypoint 101 | orb destroyed (Khalim's Will) | class 242 `mephisto`, level 26/59/87; resists Normal fire 33 light 33 cold 25 poison 50; Hell phys 20, magic 50, all elements 75; TC `Mephistoq` quest TC (V); NOT in the table: his skills | Kolbot: stand (17566,8069), sidestep to radius 18 when his mode is `Attacking2` (S Mephisto.js); the moat trick (teleport sorc only) walks (17563,8072)->(17610,8094) and lures him >27 away, then casts across the moat; optional council clear at (17600,8125),(17600,8015),(17643,8068); red portal bridge around (17590..17601,8068..8070); hydras from his casts must expire before the portal (RushThread waits on them, S). Soulstone `mss` must be picked up (Hellforge quest; planner.md V) |

Dangers per class (U unless a V column above): Sorceress - Mephisto lightning/cold at Normal is only 25-33 percent resisted, the moat trick avoids all melee/missile danger but needs Teleport (the bot has `nav` teleport hops, V live in Act 2) and `boss.Script.NeedsTeleport` points; Paladin/Barbarian melee - the `Attacking2` sidestep matters, Normal has no immunities (V) so any build works; Amazon/Assassin poison skills halve against Mephisto (poison 50, V); Necromancer - Corpse-based builds are fine (Council packs leave corpses); Druid fire is 33 percent resisted. Hell: lightning-immune Council members and 75 percent Mephisto resists need physical/magic damage or a merc (planner.md S).
`runs.json` floors (R): Travincal 24/48/70, Mephisto 28/55/75 (V that these are in the repo data; their sourcing is U).

## 7 Support matrix and gaps (V code)

| capability | have | gap |
|---|---|---|
| Hop A2->A3 | `quest.Hops[1]` step data, rows U | adapter, live check (Meshif "Sail East" row), then `Hops` tags |
| Act 3 quest data | none | `quest/act3.go` (area consts V above, NPC consts, item codes V, handlers as data) |
| Act 3 task nodes | none | `task/act3*.go`, campaign `NormalAct3`, reader mapping (`taskQuestIDs` only has Act 1) |
| Relic fetch legs (3 chests) | pattern exists (`a2fetch`, `openAndTake`) | reuse; Sewers 93 lever/stairs, per-level chest hop, nav hints |
| Flail from the Council | `Fighter.Kill`, loot pickup by name | script `High Council` (kill 3, loot `qf1`), match by classes 345-347 |
| Cube Khalim's Will | `town.Transmute` (2 inputs live) | 4-input run, layout test, result `qf2` must stay in the pack, unequip guard |
| Orb smash | none | equip `qf2` from the pack (only `hand.Unequip` exists), left-click object 404 until its mode changes, re-equip the old weapon (`town.EquipForTest` is a test hook) |
| Mephisto script | framework only | `boss/act3.go` with Match class 242, WP 101, Route {102}, stands, `Attacking2` hazard (needs monster mode in `boss.Mon`, U), optional moat phase for teleporters |
| Red portal hop | `Hops[2]` with `ObjRedPortal = 60` | verify object (342?), appears only in the same game as the kill (S Kolbot, the portal is gone after leaving) |
| Merc (Asheara) | `town.MercWant`, hire window live-probed for Act 2 | rows for Act 3 (V rows by seller 252), `Act3Merc` data |

## 8 Work list (small packages for d2-worker unless noted; "live" = d2-live after the offline part)

Shared (X) first, they unblock every act:

- **X1 campaign templates**: add `normal_act2`, `normal_act3`, `normal_act4`, `normal_act5`, `normal_all` to `goals.CampaignQuests` and `internal/bot/runner.go` (today only `normal_act1`); entries counts must equal `len(Entries)`. Offline test: template names plan and finish with the fake env.
- **X2 boss runner wiring**: `cmd/d2bot/boss.go newBossRunner` uses `boss.All()` (extended with Acts 3-5) and resolves super uniques with `db.SuperUniqueByName` (`WithUniqueNo`) for the Council six, Shenk (42), Nihlathak (60), Hephasto (41), the Ancients (43-45). Offline test: every name in `task.Boss*` has a script.
- **X3 d2bot task adapter for Acts 2-5**: `taskQuestIDs` (map the Act 2 ids that already exist in docs/act2.md plus the new ones), `npcClass`/`talkPaths` for Atma..Qual-Kehk, `Interact(object)` names, `task.Transmuter`, `StallRecoverer`, `MercHirer`. Largest package; split per act once the node ids exist.
- **X4 nav tables**: `act3Hints`, `act4Hints`, `act5Hints` (edge crossings by waypoint order, S/U, confirmed by first ExitTo), `knownWaypointPos` for 75/103/109 after a live reading, wpPanel tab x for acts 4/5 (U) and the icon column for rows >= 7 in acts 3 and 5. Offline: `nav` sim tests for the hint graph; live: panel clicks.
- **X5 quest weapon equip**: `hand.Unequip` exists (V live, docs/act2.md T0) but nothing equips from the pack except the test hook `town.EquipForTest`; add `hand.Equip(code)` (pack -> weapon slot) and a `town`/`task` capability `UseWeapon(item, fn)` that swaps in a quest weapon, runs fn (orb smash, forge hit) and swaps back. Offline against the sim, live check on a cheated character.
- **X6 quest item pickup**: extend `loot.unwear` and the pickit "always take" list with `j34 g34 d33 g33 bbb qey qhr qbr qf1 qf2 mss hfh ice tr2 xyz luv`; test with fakes.
- **X7 task World fields**: `CanTeleport`, `HasMerc`, `NoTownPortal` through `bossActs` (already requested in docs/act2.md); constraint test: Ancients plan never emits TownPortal (planned in tasklayer.md).

Act 3 packages:

- **A3-1 quest data**: `internal/quest/act3.go` (area ids V, NPC classes V, item codes V, objects, `Act3Order`, handlers) + `act3_menus.go` rows for Hratli/Alkor/Ormus/Asheara/Cain with tags U; a `TestAct3ConstantsAgainstRealTables` like Act 2 (skips without `/tmp/d2mpq`).
- **A3-2 task nodes**: `internal/task/act3.go`: `QKhalim`, relic legs `eye/heart/brain`, `flail` (Council), `khalims_will` (cube + equip + orb), `golden_bird`, `blade`, `lam_esen`, `blackened_temple`, `mephisto`, `act3_hop`, `merc_act3`; `NormalAct3(opts)` with floors (Council 24, Mephisto 28 from runs.json) and `NormalActs123`. Offline tests like `act2_test.go`.
- **A3-3 boss scripts**: `internal/boss/act3.go`: `High Council` (anchor Boss, three matches via a new multi-match field or three scripts), `Mephisto` (above). Tests with `fake_test.go`.
- **A3-4 live (d2-live, instance named by the main agent)**: cheat a clvl-28 Normal character into Act 3 (`d2cheat act N 3`, waypoints all), confirm the table above: areas, which chest holds which relic, Council positions, orb mode change, the hop and `ObjRedPortal`; one dated line each in docs/progression/results.md.
- **A3-5 Sewers and Durance walk**: nav live pass for 80/81 -> 92 -> 93 (lever/stairs) and 83 -> 100 -> 101 -> 102 (stairs). Stall recovery already exists (`StepTravelA2`): generalise it to a `StepTravelA3` or a shared `StepTravel` with the recoverer.

## 9 Open questions (each needs a live check or a better source)

1. Which monster drops the Jade Figurine and which flow gives the Gidbinn (U; candidates in 2).
2. Whether the Lam Esen altar works before the Blade quest (Kolbot says yes in rush mode, S; Arreat says the quest is given after it, V text).
3. Chest class to relic mapping (405/406/407): S Kolbot only; the unit's chest content is read after opening (`loot` by name `Khalim's ...`).
4. Whether the Flail comes from a fixed member or any (all six share TC "Council", V; item entry inside it not dumped).
5. Mephisto's skill list and the hydra source are not in our dump (the Monster struct drops `Sk*` columns); a table read would settle the `Attacking2` hazard design.

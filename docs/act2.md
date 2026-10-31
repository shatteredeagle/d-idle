# Normal Act 2 (agent ACT2, offline)

New files only: `internal/quest/act2*.go`, `internal/boss/act2*.go`, `internal/task/act2*.go`, this file. Nothing was run against a
live game. Tags: V verified (against the real tables on the worker host, `go test -run Act2Constants ./internal/quest`, which re-checks every V id),
S secondary (Kolbot, Arreat), U unverified.

## What exists

| layer | file | content |
|---|---|---|
| quest | `act2.go` | area ids (V), NPC classes (V), object classes, quest item codes (V), `Act2` handlers as step data, `Act2Order`, `Act2Proxy` (item proofs for the fetch legs), `Act2Merc` |
| quest | `act2_env.go` | `Act2Env` = `Env` + `Cube` + `UseItem`, `PlaceItem`, `OpenChest`, `HasObject`, `HireMerc`; `Act2Runner` (`Run`, `RunAll`), `FindTomb` |
| quest | `act2_cube.go` | the `Cube.Transmute(inputs)` contract, the panel UI steps, 3x4 grid maths (`PlanCubeLayout`) |
| quest | `act2_menus.go` | dialog rows of Atma, Drognan, Fara, Elzix, Lysander, Geglash, Greiz (V), Warriv A2, Cain A2, Tyrael, Meshif A3 (appended to `Menus` in `init`) |
| quest | `act2_fake.go` | `FakeAct2Env` for tests |
| boss | `act2.go` | scripts `Radament`, `The Summoner`, `Duriel`; `All()` (Act 1 + 2); `ColdAura` hazard; `RunWithEntry` (enter the lair first) |
| task | `act2.go`, `act2_campaign.go` | nodes for radament, horadric_cube, viper_amulet, staff_of_kings, horadric_staff, arcane_sanctuary, summoner, duriel, act2_hop, merc_act2 (registered in `init`); `NormalAct2`, `NormalActs12` |

Tests: `go test -p 1 -count=1 ./internal/quest ./internal/boss ./internal/task` (the head box, per package). Real-table check on the worker host:
`rsync` the worktree to `/tmp/act2wt` and `go test -count=1 -run Act2Constants ./internal/quest`.

## Quest flow as coded (Normal)

1. **Arrival** (A1 -> A2 is the existing `Hops[0]`, Warriv class 155, arrival area 40 V): the hop ends with `KTouchWP`. Then Jerhyn (`act2_intro`, word 8).
2. **Radament's Lair** (word 9): Atma 176, Sewers 2 waypoint (48), Sewers 3 (49), kill Radament (class 229, super unique 10, V), pick up the Book of Skill
   (`ass`, V), right-click it (+1 skill point), open the Horadric Scroll chest (object 355, S), take the scroll (`tr1`), town, Atma, Cain reads the scroll (U).
3. **Fetches** (no turn-in; the item in the pack is the proof, `Act2Proxy`):
   - Cube `box`: Halls of the Dead 3 (**60**, V), chest 354 (S). Waypoint Halls of the Dead 2 (57).
   - Amulet of the Viper `vip` (display "Top of the Horadric Staff", V): Claw Viper Temple 2 (**61**, V), altar object 149 (V "taintedsunaltar"), guard Fangskin (super unique 12, V). Lost City 44 -> Valley of Snakes 45 -> temple 1 (**58**) -> 2.
   - Staff of Kings `msf` (display "Shaft of the Horadric Staff", V): Maggot Lair 3 (**64**, V), chest 356 (S). Far Oasis 43 -> lair 1 (62) -> 2 (63) -> 3.
4. **Assemble** in the Cube: `msf + vip -> hst` (V: cubemain.txt row "Staff of Kings + Viper amulet -> Horadric Staff"). Done in town.
5. **Tal Rasha's tomb**: Canyon of the Magi (46), seven tomb levels 66..72, exactly one has the orifice object 152 (V "orifice"). `FindTomb` walks them and uses
   `HasObject` (presets). Place the staff, then object 100 (V "Duriel's Lair") leads to area 73.
6. **Arcane Sanctuary / Summoner** (optional): Drognan, Jerhyn, Palace Cellar 3 (54), portal object 298 (U), area 74 (V), kill The Summoner (class 250, V), journal object 357 (V exists as "Tome", S that it is the journal), red portal 60 to the Canyon (U).
7. **Duriel** (class 211, V): kill in 73, Tyrael (class 251 "tyrael1", V) talk, town, Jerhyn, Meshif (210, V) "Sail East" to Kurast Docktown (75, V) = existing `Hops[1]`.
8. **Mercenary** (`Act2Merc`, `merc_act2`): Greiz (198, V live) Talk / Hire / Cancel; Normal rows Prayer (Combat, Id 6), Defiance (Defensive, Id 7), Blessed Aim (Offensive, Id 8), V hireling.txt. Default Defiance.

### Corrections to older docs (V from levels.txt)
`docs/goals/execution.md` / `catalog.md` list Halls of the Dead 3 as 58 and Claw Viper 2 as 61 and Maggot Lair 3 as 64 "from memory". The table says
Halls of the Dead 3 = **60**, Claw Viper Temple 1 = **58**, Claw Viper Temple 2 = 61, Maggot Lair 3 = 64. `quest.Meshif` (210) is the Act 2 harbour Meshif; the Act 3 one is 264
(`MeshifA3`). Act 2 Warriv is 175 (`WarrivA2`, already present). Duriel's Lair has `Teleport = 2` in levels.txt (V): no teleporting inside.

## Horadric Cube primitive (request for `internal/town`)

Contract: `quest.Cube` (`Transmute(ctx, inputs []string) (outputs []string, err error)`; task layer twin `task.Transmuter`). Proposed home: `internal/town/cube.go`,
`func (t *Town) Transmute(...)` on top of `town/hand` Take/Put and the panel reader, next to `Stash`. Facts: Cube is a 2x2 item (V), its grid is 3 columns x 4 rows of 29 px
(V inventory.txt "Transmogrify Box Page 1"; grid origin on screen U: the table's gridTop 139 lies outside the panel band 220..442, so calibrate from a screenshot).
Item sizes (V): msf 1x3, vip 1x1, hst 1x4. `PlanCubeLayout([]Size)` gives the cells (first fit, tallest first, tested). UI steps are in the comment of `act2_cube.go`.
On failure the implementation must give the inputs back to the pack.

## Live checks (for a later session; one dated line each into results.md)

Throwaway character on a worker host instance, cheated to the state named, standing in Lut Gholein unless stated. Take a screenshot with `tools/shot.sh` where marked.

| id | check | expected | screenshot / read |
|---|---|---|---|
| A1 | Walk Warriv A1 -> area 40 after Andariel (existing hop) | `Player.Area == 40` | row index of "Go East" |
| A2 | `townlive units` in Lut Gholein | classes 176 Atma, 177 Drognan, 178 Fara, 198 Greiz, 199 Elzix, 200 Geglash, 201 Jerhyn, 202 Lysander, 175 Warriv, object 156 waypoint; which Cain class stands there after the rescue (244/245/246/265/520) | positions only as a sanity check, never as constants |
| C2 | Talk menus of Atma, Drognan, Jerhyn, Cain, Tyrael: row count and labels, before and after the quest | Talk / Trade / Cancel plus quest topics; fix `act2_menus.go` rows and flip tags | screenshot of each open dialog |
| R1 | Radament: Sewers 3 = area 49; waypoint at 48; boss monster class 229, super unique no. 10; the drop is `ass`; the scroll chest object id (355?) and `tr1` | as listed | pack dump, `Presets` of area 49 |
| R2 | Right-click the Book of Skill | stat 5 +1, item gone; word 9 after Atma's second talk = 0x0001/0x8001 | character screen |
| F1 | Halls of the Dead 3 (60): chest object (354?) gives `box` | `box` in the pack | pack dump |
| F2 | Claw Viper Temple 2 (61): altar 149, Fangskin; `vip` drops from the altar click, not from Fangskin | `vip` after the click; word 11 | pack dump |
| F3 | Maggot Lair 3 (64): chest (356?) gives `msf` | `msf` | pack dump |
| C1 | Cube UI: right-click the Cube in the pack; record the panel flag, the pixel origin of the 3x4 grid, the Transmute button position; run msf + vip | `hst` appears in the cube grid; both inputs gone | screenshot of the open panel, then of the result |
| T1 | Tal Rasha's tomb: stand in the Canyon (46), enter tombs 66..72, call `HasObject(152)` per level | exactly one level answers true; objects only from presets | list of areas that answered; count of tombs entered |
| T2 | Place the staff: item on cursor then click the orifice, or click with the staff in the pack | object mode changes, object 100 appears, word 10 = Done | screenshot before/after |
| T3 | Object 100 leads to area 73 | `Player.Area == 73` | area read |
| S1 | Palace Cellar 3 (54): portal object 298; Arcane Sanctuary 74; which waypoint object class is the Arcane one (323?) | arrival in 74, waypoint touchable | screenshot of the portal |
| S2 | Summoner: preset 357 offset (-3,-3) vs where he stands; class 250; journal 357 click opens a red portal (object 60) to the Canyon | portal object, arrival 46 | positions of preset and boss |
| D1 | Duriel: Normal stats from the table (life 4757, cold resist 50) vs observed; `Teleport = 2` in the lair; where `ColdAura` should fire (radius 7, life < 55%) | no teleport; the chill is visible | screenshot at melee range, life readings |
| D2 | Tyrael (251) appears after the kill; talk row; word 14 | word 14 Done after Tyrael, word 15 after Jerhyn | dialog screenshot |
| H1 | Jerhyn talk then Meshif "Sail East" row (existing hop) | `Player.Area == 75` | row index |
| M1 | Greiz hire with `Act2Merc` (aura Defiance): `town.MercWant{Act: 2, Diff: Normal, Aura: "Defiance", Row: n}` | gold drops, merc unit appears | hire window screenshot |

## New bot capabilities needed (none implemented here)

1. **Cube UI**: `Transmute` (see above), including reading the cube container's items.
2. **Staff assembly**: only a `Transmute([msf, vip])` call plus a Cube-in-pack check; the handler exists.
3. **Tomb opening**: `FindTomb` needs `HasObject(class)` answered from presets of the current level (static map), and `PlaceItem(code, objClass)` (pick to cursor, click the object; fallback click with the item in the pack, U). Task layer uses `Interact("tal-orifice")`, and the adapter does the scan, the placement and sets the pseudo item `Horadric Staff placed`.
4. **Summoner portal**: Palace Cellar 3 portal (object 298 U), Arcane waypoint, journal click and the red portal; a Teleport check for the Arcane maze (no capability flag exists in `task.World`).
5. **Chest/altar open + pickup** as one adapter call (`OpenChest`, task `Interact("<object>")` followed by `PickUp`); quest chests are pickit-exempt.
6. **Book of Skill use**: right-click an item in the pack (`UseItem`, task `Interact("item:Book of Skill")`), set the pseudo item `Book of Skill read`.
7. **Mercenary**: `MercHirer.HireMerc(ctx, act, aura)` over `town.Hirer.Hire` (needs the hire-window row from a screenshot reader, U).

## Requests for INTEGRATE2 and other owners (existing files, not edited here)

- `internal/boss/act1.go`: `Lookup` and `Runner.Scripts == nil` only know Act 1. Use `Runner{Scripts: boss.All()}` (or make `Act1()` merge `Act2()`). Duriel needs `boss.RunWithEntry(ctx, script, w, a, opts, enter)` with `enter` = the tomb leg; the plain `Runner.KillBoss` has no entry hook (add one field `Enter func(ctx) error` and call `RunWithEntry`).
- `internal/task` `World`: add `HasMerc bool` (today the pseudo item `Mercenary` stands in), `CanTeleport bool` (Summoner, Duriel gating), `Bosses killed` is per game (restart loses it; the quest reader fixes that for the quests that have a word).
- Pseudo items the adapter must maintain in `World.Items`: `Horadric Staff placed`, `Book of Skill read`, `Mercenary`. Item names are the game's display names (V items.txt Display): `Horadric Cube`, `Shaft of the Horadric Staff`, `Top of the Horadric Staff`, `Horadric Staff`, `Book of Skill`, `Horadric Scroll`.
- `cmd/d2bot/task.go` `saveQuestReader`: map task ids to save ids: radament -> radament, horadric_staff -> horadric_staff, arcane_sanctuary -> arcane_sanctuary, summoner -> summoner, duriel -> seven_tombs, act2_complete -> act2_done. The fetch legs (horadric_cube, viper_amulet, staff_of_kings) have no word of their own (the Tainted Sun word 11 is `tainted_sun`, which can also back `viper_amulet`). `talkPaths`: Atma "quest"/"reward", Cain "horadric scroll", Drognan/Jerhyn "arcane", Jerhyn "act2", Tyrael "reward", Meshif "sail east" (the existing "Sail East" row).
- `task.Actions`: optional interfaces `task.Transmuter` and `task.MercHirer` on the same value; objects the adapter must resolve by name: `Horadric Scroll chest`, `Horadric Cube chest`, `taintedsunaltar`, `Staff of Kings chest`, `tal-orifice`, `duriel-door`, `Arcane Sanctuary portal`, `Summoner's journal`, `item:<name>`.
- `internal/quest/dialog.go` (INTEGRATE2 owns it): constants `Atma` etc. live in `act2.go` of the same package; `Menus` gets the Act 2 rows from `init` in `act2_menus.go`, so do not add duplicate tables.

## Dead ends / not done

- Whether the Horadric Scroll chest is in Sewers 3 and whether Cain's talk starts the Staff quest is S (execution.md V from Arreat, Kolbot object ids); the order of rewards vs words is untested.
- No tomb identification shortcut (Summoner journal reveals it in the original game, S): the bot scans.
- Hazard numbers (Duriel cold aura radius 7 / life 55 %, kite points) are guesses (U).
- Nightmare/Hell Act 2: floors in the scripts follow the Summoner spacing (50/70) and are U; `NormalAct2` is Normal only.

## Live results (ACT2LIVE, Navi on i1, GE-Proton, 1.14d, 2026-10-29)

Setup (all logged cheats, reason "ACT2LIVE ..."): `d2cheat save Navi level=24 gold stashgold skills=... maxlife=...` and `act Navi 2` with the game at the
main menu (OOG leave, edit, OOG start), then Navi stands in Lut Gholein. A leftover god-mode `maxlife` (2097151) from an earlier agent had to be reset first
(`save Navi maxlife=60`, then `level=24`: the level edit adds on top of the stored value, and a value over 21 bits is refused).
Tool: `tools/rigs/act2live` (`state units presets vis go hunt kill obj pick use goto clear`, chain with `--`, `god SECS` first = logged invincibility for the run;
a level 24 Sorceress with Fire Bolt 12 dies at once in the sewer packs without it). Tags: V live, U not seen.

| id | result |
|---|---|
| A1 | V: `act Navi 2` loads Lut Gholein, `Player.Area == 40`, Warriv A2 (175) stands at the spawn. (The Warriv hop itself was not walked.) |
| A2 | V (units around the town, range limited): 175 Warriv (5162,5202), 176 Atma (5140,5053, on the tavern porch), 178 Fara (5118,5081), 201 Jerhyn (5060,5146), 202 Lysander (5115,5109), 244 an NPC next to Fara/Atma at (5134,5085) which is Cain after the rescue (class 244 = `CainA2`, now V that the class exists in town), 195/196 guards, 203/204/331 other town NPCs. Drognan, Elzix, Geglash, Greiz not in range of what was walked. Object 156 (waypoint) not seen; presets list objects 121, 285, 286, 287 (not waypoint). |
| C2 | V: Atma menu = Talk / Cancel (no Trade). Clicking Talk plays the Radament introduction at once (a speech box, no submenu) and her walk-off; word 9 = 0x0004 afterwards (`d2qmem`). Rows fixed in `act2_menus.go`. Other NPC menus not opened. A first `townlive menu 176` right after loading failed ("pointer never hovered") while the intro speech overlay was up; a retry worked. |
| R1 | V: route 40 -> 47 -> 48 -> 49 works (nav exits). Sewers 3 (49): Radament class 229 at (7837,8023), the monster preset id 744 at (7837,8018); he stands inside a huge pack (dozens of Burning Dead / Burning Dead Mage / Dried Corpse: lethal below god mode). `Fighter.Kill` killed him (event boss=true, xp 2234). Drop: Book of Skill `ass` (class 552, display "Book of Skill" with the "Right Click to learn skill of your choice" line) plus gold, potions, gems, a key; loot.PickUp with Names {"Book of Skill"} took it. Chest object 355 at (7832,8027) is NOT in `Presets` (only present once the area streams in) but is in the object table next to Radament; clicking it with `Interact` drops the Horadric Scroll `tr1` (class 550, loot picked it by name). Object mode stays 0 after opening. |
| R2 | V: right-click `ass` in the pack (hand.RightClick, pack open): skill points 23 -> 24 (`d2cheat show`), item gone. Quest word after this not read. |
| nav | V finding: `navlive route 40 41` = "no route": levels.txt Vis of 40 only lists 47 and 50; all outdoor Act 2 links are level-edge crossings. Added `internal/nav/act2hints.go` (40-41-42-43-44-45, S). |
| F1 | V: chest 354 at (15363,12623) in Halls of the Dead 3 (60), `Presets` shows it only after the room streams in (the chamber is east of the entrance corridor, beyond a wall that nav's walk grid does not path through; reached by Teleport casts `game.CastAt("f2", point)`, each moves about 18 subtiles). Opening it with `Interact` drops the Horadric Cube `box` (class 549, 2x2) with gold, potions and a charm; `loot.PickUp{Names: {"Horadric Cube"}}` took it once the pack had a free 2x2 (a full pack gives "no room"; junk was dropped with Take + a left click on the ground). The chamber holds a boss pack (one boss, 6 kills, 1845 xp with god mode). Chest object ids differ every game (1223, 287, 294), never use ids. |
| C1 | V (screenshot + code): right-click the Cube in the pack opens the cube panel on the LEFT half, flag 0x1A (`d2.UICube`) with 0x01 still set. `panel.Layout.Cube` = left 198, top 199, 3 x 4 cells of 29 px matches the screenshot; `hand.Take` from the pack and `hand.Put` into the cube at (0,0) both verified (`Item.Where == d2.ContCube`, cell (0,0)); taking it back and `CloseAll` leave the pack as before. The Transmute button is the icon under the grid at about (240,328) client pixels (screenshot); clicking it was NOT run (no recipe items at hand), so the Transmute click and the result placement stay U. The close button of the cube panel is at about (370,450). |
| nav2 | V: the Lut Gholein waypoint is object 156 at (5069,5084) (stash 267 at (5121,5073)); it is absent from `Presets` until the west part of the town has streamed in, so `TakeWaypoint` from town said "none on level 40" right after loading and works after a walk across the town. With all waypoint bits set (`d2cheat waypoints Navi all`, logged), `TakeWaypoint(57)` from Lut Gholein worked and saved the desert trek. `nav.ToArea` through Rocky Waste / Dry Hills with hints works but needs 5+ minutes and stalls inside monster packs ("nav stuck" rungs 0-5, `combat move failed`); Halls of the Dead 3 is the same story beyond x=15305. |
| objs | V objects.txt names: 100 "Duriel's Lair", 149 "taintedsunaltar", 152 "orifice", 298 "portal" (act 2), 354/355/356 "chest" (token xk, act 15), 357 "Tome", 60 "Portal", 156 "Waypoint" (token wm, act 2). |

Not done (no time left in this session; every step below is still U): F2 (Claw Viper altar), F3 (Maggot Lair chest), staff assembly (Transmute click), T1-T3 (tomb search, orifice, area 73), S1/S2 (Summoner portal 298, journal), D1/D2 (Duriel, Tyrael), H1, M1, the Cain read of the Horadric Scroll (the scroll is in the pack), the other NPC menus (C2 for Drognan, Jerhyn, Cain, Fara, Greiz, Elzix, Geglash).

### What this session says about the flow
- Dying is the main cost: a level 24 Sorceress with Fire Bolt 12 dies to every Act 2 pack the moment the logged god cheat ends (jobs end god at their timeout, so chain every leg of a trip into ONE `act2live` run). A bot run needs real combat strength before Act 2, or Teleport-first routes that avoid packs.
- The quest chests (354, 355, 356) are not preset-visible from afar: `FindTomb`-style "ask presets" logic works only inside the streamed rooms. Chest positions are fixed per level layout (Sewers 3: next to Radament; Halls 3: far east chamber), but the layout is random per game, so a bot must walk or teleport there and look at the object table.
- Teleport with `CastAt` is the fastest and most reliable mover in these levels; `nav` should learn to use it (not done here).

### Nav fixes after the live runs (PORTALNAV)
GoTo avoids exit tiles that are not its target (Maggot Lair entry on the way to the Far Oasis waypoint) and nav has a `Yield` hook plus a retreat rung for pack stalls; details and tags in docs/nav.md. The Halls of the Dead 3 chamber wall is unchanged (U, needs a live map dump).
## Live results 2 (ACT2LIVE2, Navi on i1, 2026-10-29)

Chain tool additions in `tools/rigs/act2live`: `exitto LEVEL` (clear, then teleport hops to the exit / adjacent contact, then ToArea; explores otherwise), `tpto X Y`, `cubemix CODE...`,
`unequip X Y`, `invshot SECS`, `adj`. Every leg ran under `god SECS` (logged cheat, reason "ACT2LIVE: ...").

| id | result |
|---|---|
| F3 | V: Maggot Lair 3 (64): chest object class 356 (id differs per game) next to the level's far end; `Interact` drops the Shaft of the Horadric Staff `msf` (class 92, 1x3) with a magic Breast Plate and gold. Route from Lut Gholein via waypoint 43 worked once the Far Oasis was reached; Maggot Lair 1 is a corridor full of Sand Maggots that blocks nav completely: `combat.Clear` (not nav) is what clears it (26 kills, 2 bosses). Exits found by `Exits` (Dest 63, 64) then teleport hops + ToArea: 62 -> 63 -> 64 took about 3 minutes. |
| F2 | V: Claw Viper Temple 2 (61), object class 149 (taintedsunaltar) at the far end; its pack (class 75 Claw Vipers) blocked `GoToUnit`, a `Clear` first fixed it. `Interact` on the altar drops "Top of the Horadric Staff" (`vip`, quest unique, class 521) that `loot.PickUp{Names}` takes (display name is NOT "Amulet of the Viper"). Route: Valley of Snakes (45) exit Dest 58 (temple 1), temple 1 exit Dest 61. |
| P1 | `PlaceItem` rewritten: GoToUnit next to the orifice, `findHover` (scan 13x11 screen points around the projected position for one where `Hover` reports the object; V: 4..51 hits at the orifice), take the Staff to the cursor (hand.Take), close panels, click the found point. V: with an item on the cursor `Interact` (hover check) never sees the object ("pointer never hovered"), so the point must be found before taking the item. V: a blind click at the projected position (376+16*(dx-dy), 270+8*(dx+dy)) missed the orifice and the Staff fell on the floor. U: the final click (never reached with the Staff). |
| P2 | V: `grab hst` (Interact with the ground item) puts the Staff straight into the pack (not on the cursor). `takeitem hst` takes it onto the cursor; any walking click (GoToUnit, Clear, nav) with the item on the cursor drops it. V: `OpenInventory` next to the orifice failed several times with "ui flag 0x1 never became true" (ui flag 0xE set after walking up to the orifice; U what it is); `key i` before touching the orifice works. |
| P3 | V (loss): a Staff on the cursor when Navi died (god ended between runs) was gone afterwards (not in the pack after the respawn, no ground item seen near the death spot with `units 0`). Never keep the cursor item across a run end. A restart from the save restores it. |
| P4 | NOT reached: the orifice accepting the Staff (so no object 100, no area 73, Duriel, Tyrael). |
| T0 | V: clicking the slot again with the item still on the cursor re-equips it (the first unequip attempt looked like a no-op for that reason). |
| C3 | V: Transmute works: cube open (flag 0x1A), `Transfer` msf and vip into the cube (cells (0,0) and (1,0)), click (240,328) once, 2.5 s later the cube holds `hst` "Horadric Staff" at (0,0); `Transfer` out into the pack worked. The quest word was not read. Implementation in `tools/rigs/act2live` `cubemix` (hand.Transfer both ways); `town` still needs the real `Transmute` built from this. |
| W1 | V: the waypoint panel of Lost City shows ONLY the visited waypoints active: the `waypoints all` cheat from the first session was gone (Navi is reloaded between sessions, and the cheat is a save edit). `TakeWaypoint(46)` -> "level 46 not reached". Re-run `d2cheat waypoints Navi all` at the main menu before a long trip. |
| nav3 | V: Far Oasis, Lost City and Valley of Snakes are chained by `Adjacent` contacts (44 contact X4120 / X4520, 160 subtiles tall): teleporting to the contact then `ToArea` crosses in seconds; plain nav stalls for minutes inside the cave-like packs (identical "nav stuck" at one subtile for 5+ minutes while monsters block: `Clear` frees it). Valley of Snakes (45) rect X3960..4120 Y4040..4200 is a dead end (only 44 adjacent, exit Dest 58); the Canyon (46) is NOT reached from it (west teleport blocked by rocks). `Adjacent` lists only rooms already streamed (43 did not list 42), so absent entries prove nothing. |
| god | V finding (cheat tool): killing an `act2live` job (SIGTERM before `god stop`) leaves max life at the 8191 buffer and later runs take that as the "original": Navi now has 8191 max life. Restore with `d2cheat save Navi maxlife=..` at the main menu. |

Not reached: T1-T3 (Canyon / tomb search, orifice, area 73: the Canyon of the Magi was not reachable without its waypoint; the route from the desert chain to 46 is still U), S1/S2, D1/D2, H1, M1, Cain reading the scroll.

## ACT2CODE: offline implementation of the Live results 2 findings

Tags: V = verified live in `Live results 2` or from the 1.14d tables, S = secondary, U = unverified. Fake-tested only; nothing here ran live.

- **Cube**: `town.(*Town).Transmute(ctx, inputs) ([]string, error)` (internal/town/cube.go) satisfies `quest.Cube`; inputs may be base codes or display names, so `task.act2` calls it directly (`task.CubeOut`). Flow V (cubemix): open pack, right-click the Cube, wait flag 0x1A, `hand.Transfer` in, click (240,328) at 800x600 (stored as base (160,268) + layout offset, V only at 800x600), wait up to 3 s for a new item, Transfer out, `CloseAll`. Failure empties the cube back into the pack. Worn inputs are taken off first (`hand.Unequip`: click slot, `ClearCursor`; slots rArm/neck V, others U). The sim got a cube, the msf+vip recipe and unequip so the tests run offline (needs /tmp/d2mpq).
- **loot**: after a pickup of msf, vip, hst, box, tr1, ass the item is taken off the body if the game wore it (`Looter.unwear`). The game still equips on click (P1); we undo it.
- **nav.TakeWaypoint**: with no waypoint unit and no preset, walks to `knownWaypointPos[level]` (Lut Gholein 40: (5069,5084) V, may shift by seed U), then retries; fails if it is already there and nothing showed up.
- **act2hints**: 45-46 hop (U). levels.txt Vis of 46 lists only the seven tombs (V, dumped), so no data-backed outdoor link exists; the waypoint is the dependable route.
- **Stall recovery**: `task.StepTravelA2` (all Act 2 travel nodes) on stuck/timeout/not-found calls the optional `task.StallRecoverer` (`TeleportToward` first, then `ClearAround`) and retries twice. The bot's Actions must implement it (d2bot adapter: ClearAround = combat.Clear around the player, TeleportToward = the `exitto` logic of tools/rigs/act2live); without it behaviour is unchanged.

## Live results 3 (ACT2LIVE3, Navi on i1, 2026-10-29)

New `tools/rigs/act2live` pieces: `rig` (implements `task.StallRecoverer`: ClearAround = `combat.Clear`, TeleportToward = exit/Adjacent hop then `ToArea`, `ErrNoTeleport` when no exit is known; `exitTarget` is unit-tested), `a2env` (real `quest.Act2Env` subset: `GoArea`, `HasObject` from presets plus units, `PlaceItem` = walk + click), commands `findtomb`, `place`, `npc`, `click`, `talk`, `key`, `esc`, `npcgrid`; `obj` hops to the preset when the object is not streamed.

| id | result |
|---|---|
| X1 | V: the save-edit cheats (`waypoints`, `save maxlife=`) need the game stopped and an instance cannot be restarted by an agent, so they were not possible mid-session. Max life was restored through the memory cheat instead: `d2cheat setstat stat=7 value=74240` then `stat=6` (24.8 fixed point, 290 life), logged. |
| X2 | V finding (god): `tools/jobs.sh kill` of an `act2live god` job leaves max life at the 2097151 buffer every time. After every kill run the two setstat lines of X1. A god run that ends by itself restores it. |
| X3 | V: Teleport does not work in town (mode 5 neutral); `tpto` hops just print and the character stays. Town legs must walk. |
| X4 | V: palace gate: Kaelan/guards say "You may not pass" until Drognan has told the Arcane Sanctuary lead (walking up to him opens the speech by itself; a click on him continues it) and Jerhyn has spoken (same). Afterwards `exitto 50..54` (Harem 50 -> 51 -> 52 -> 53 -> Palace Cellar 3 = 54) worked under god in about 3 minutes. |
| X5 | V: `Interact` on a town NPC fails with "pointer never hovered the target" when the NPC is far from the screen centre or the character stands on stairs; a plain left click on the NPC's body opens his menu (Jerhyn: "Talk / Cancel", Talk opens Introduction, Gossip, Tainted Sun, The Seven Tombs, About the Merchants). The `Interact` hover check needs a fix (town/hand owner). |
| X6 | V: Palace Cellar 3 (54) preset list holds object 298 ("portal") at (10073,8668), unit class 176 "chest" objects around it; the 298 unit is NOT in the unit table until about 80 subtiles away (far preset), so `HasObject`/`UseObject` must hop to the preset first. |
| X7 | V (danger): Navi without gear (clvl 24, 290 life) dies in Act 2 within seconds when god ends or is not active; "You have died / press ESC" (`key esc`) respawns in Lut Gholein, gold is dropped on the corpse (31141 -> 0, 10228 toll). The corpse with the gold lies in the Rocky Waste (5096,4940). |
| S1 | V: Palace Cellar 3 (54) object class 298 "portal" (id 474, mode 2 = open, at (10073,8668) = its preset) entered by a plain click on the portal (`act2live click 390 215` while standing next to it) leads to Arcane Sanctuary (74, level rect X25000 Y5000 1000x1000, arrival (25428,5428)). `Interact` (hover check) from `obj` did not enter it; the click did. Arcane waypoint is object class 402 at (25449,5449): `obj 402` activated it (mode 0 -> 2). The red portal / Canyon leg stays U. |
| S2 | NOT reached: The Summoner (class 250) was not found. `hunt 250` ran 10+ minutes of nav.Explore in the sanctuary (platform maze surrounded by void): the Explorer sat at one subtile (25499,5321) for 5+ minutes inside Hell Clan (class 56) / Ghoul Lord / Specter packs ("nav stuck" rungs 0..5 in a loop), `combat.Clear` returned 0 kills, and Navi died twice even under `god` (X7). Needs: a teleport-aware explorer for the Arcane maze (hop across the void between platforms, the four arms end at the Summoner), or a stronger character. |
| T1 | NOT reached (Tomb / Duriel / Tyrael): the Canyon is only reachable by waypoint (save cheat impossible with the game running) or via the Summoner's journal portal (S2). `findtomb` and `place` are built and compile but never ran live. |

Open for the next live session: stop the game, `d2cheat waypoints Navi all` (and `save Navi maxlife=290`), restart i1 (main agent), then `act2live god 1700 wp 46 -- findtomb -- place hst 152 -- ...`. Gear: Navi wears nothing; put on the staff-less starter gear or cheat a level-30 character before Act 2 packs.

## Live results 4 (ACT2LIVE4, Navi on i1, 2026-10-29)

`tools/rigs/act2live` now sets `nv.TeleportKey = "f2"` (`syncTele`: off in town 40 and in Duriel's lair 73); the rig's `TeleportToward`/`reach` use `ToArea`/`Explore` with the key instead of its own `tpTo`. New subcommands: `clickobj`, `explore SECS [HX HY]`, `summoner [W E N S]`, `blindtp`, `map [STEP]`, `npci`, `hold`; first argument `debug` turns on slog debug (live.sh forwards no env). `die` now stops god before `os.Exit` (before, a failing subcommand left max life at 2097151; V, repaired with `d2cheat --reason ... --actor act2live restore`).

| id | result |
|---|---|
| Y1 | V: nav teleport explorer in the Cellar/Basement levels: `exitto 50..54` under god took about 3.5 minutes with hops of 8-12 subtiles ("nav teleport" debug lines), same as before but no manual hop code. `ExploreToward(preset)` as hint for the Cellar-3 portal (298) did NOT get there (wandered 150 subtiles away, 4 x 40 s); `obj` keeps `tpTo` for the preset hop. nav has no exported point-to-point teleport (`teleportToward` is private): wanted for the owner of internal/nav. |
| Y2 | V: portal 298 -> Arcane (74) by `clickobj 298` (plain click at screen centre while standing at the portal) works every time; `Interact` still does not. Repeated 4 times in about 3.5 min each. |
| Y3 | V: Teleport in the Arcane Sanctuary works (hops 10-17 subtiles) but `nav.Explore` stops hopping after 4 hops at (25478,5405): the hop candidate must be a KNOWN walkable cell closer to the frontier, the frontier (25539,5294) is 126 away, so it falls back to walking, which jams in Hell Clan packs ("nav stuck" rungs 0..5 loop, as in S2). |
| Y4 | V: Arcane walk-grid dump (`map 6`): the level is a set of room islands separated by void with NO room data in between (a 54-subtile unknown gap west of (25321,5438), x 25240..25294). Straight blind casts toward the Kolbot arm ends (S, from memory, U: W (25081,5446), E (25830,5447), N (25449,5081), S (25449,5812)) stop at W x=25321, E x=25578, N y=5313, S y=5586: the first void edge. The Summoner was not found; the arms need a route through other platforms (or longer hops, range above 18 is U: the screen allows more). |
| Y5 | V: the waypoint panel in Lut Gholein shows only "Lut Gholein" active (screenshot with the panel open); all other Act 2 rows are greyed although the main agent applied `waypoints all` to the save. So the cheat did not reach the running game (applied to another difficulty, wrong file, or the game kept its own copy): `TakeWaypoint(46)` fails with "level 46 not reached". The Canyon (46, levels.txt "Valley of the Kings", Vis 66..72 = the seven tombs) has no walking neighbour except via the Summoner portal, so FindTomb/Duriel/Tyrael were not reached (T1 stays NOT reached). |
| Y6 | V: `hand.InteractNPC` in Lut Gholein: opens the NPC menu when the NPC is within about 4-9 subtiles (class 201 at 9 south, class 202 at 3: menu open). Far NPCs (30-60 subtiles) and class 199 at 9 subtiles west failed ("pointer never hovered ... plain clicks opened no dialog"): the plain-click fallback does not walk far enough; call after walking next to the NPC. Which of the two paths (hover or fallback) opened the successful ones is not logged: U. |
| Y7 | V: god and kill: `tools/jobs.sh kill` of an `act2live god` job now restores max life (signal path; checked with `d2cheat show`), but the character is then unprotected and dies within seconds in the Arcane (three deaths today; ESC respawns in town, gold is 0 anyway). |

Next for this leg: either re-apply `waypoints all` with the game stopped and verify the panel (screenshot) after restart, or write the Arcane room-graph route (islands from the grid, hop between islands within teleport range) in internal/nav.

## Live results 5 (ACT2LIVE5, Navi on i1, 2026-10-29)

Tags: V live, U not seen.

| id | result |
|---|---|
| W2 | V: `waypoints all` DID reach the game (`d2qmem` reads D0 wp=0x7fffffffff; save too). The failure of `TakeWaypoint(46)`/(74) was the PANEL CLICK: in the Act 2 panel rows 0..6 (up to Palace Cellar 1) work with `wpPanel.RowX=230`, rows 7 (Arcane, y 373) and 8 (Canyon, y 407) do nothing at x=230 (also not at x=350) but travel when the row ICON is clicked (x=112, y 376 / 410). Fix needed in `internal/nav/waypoint.go` (not my directory): click the icon column (x about 112) for every row, or at least for rows >= 7. |
| W3 | V: `Interact` on the Lut Gholein waypoint (class 156) alternates between "pointer never hovered" and success; the object is absent from the unit table until the west part of town has been walked (`goto 5100 5100 -- goto 5069 5090`, then `obj 156` opens the panel). A second plain click on the object closes the panel again (toggle), so `clickobj` after `obj` closes it. `clickobj CLASS once` clicks once. `click` now waits 300 ms after the mouse move. |
| W4 | V: Act 3 row 0 of tab III worked from Palace Cellar 1 (waypoint of Spider Forest 76 reached, arrival at (5068,4128)); `wp 40` from there walks to the Spider Forest waypoint (V, found by TakeWaypoint) and returns to Lut Gholein. So the Act 3 waypoint hop exists, Kurast Docks (75) itself not tried. |
| T1 | V: `wp 46` (icon click) lands in the Canyon of the Magi at (12678,5178), mode 1, with many Devourer eggs/Steel Scarabs; `combat.Clear` + `FindTomb` worked: exits to tombs 66, 67, 68 entered and left (nav exit gates (12535,5035), (12529,5126), (12530,5206)), no orifice (class 152) in any of them (U whether the orifice room is just not streamed: only the first cave room is). The exit 69 gate (12524,5285) did not fire ("nav exit tile did not fire", then edge marked bad), so `FindTomb` aborts with "could not reach area 69". Exits seen by `adj`: 71 at (12714,5025), 72 at (12795,5032); `exitto 71` and `exitto 72` work. 70 not found. |
| T2 | V: tombs are large (level rect 1000x1000); tomb 72 arrival room is a cross of corridors (map 8). `nav.Explore` returns "nothing left to explore" right after arrival (walk grid frontier), and `HasObject(152)` only sees the streamed room, so FindTomb cannot decide per tomb. U: where the orifice is. Needs a tomb explorer that goes to the far ends (teleport / GoTo far points) before judging a tomb. |
| D | V (danger): between `act2live` runs Navi is unprotected and dies within seconds in tomb levels (twice today); a run that starts with the character dead logs "stat 6 not present". After a death `key esc` respawns in Lut Gholein (gold dropped). |

Not reached: orifice/area 73, Duriel (D1/D2), Tyrael, gold shopping (the gold stat did not exist at 0 gold; it appears after the first pickup, so `d2cheat gold` works from then on).

## TOMBFIND (offline, 2026-10-29)

Tags: V verified, S secondary source, U unverified.

- Waypoint panel (`internal/nav/waypoint.go`): `wpPanel` got `IconX=112`, `IconDY=2`, `IconFrom=7`. Rows >= 7 click the icon column first (V live, ACT2LIVE5: rows 7/8), the other rows the text column (V rows 0..6); the other column is the fallback while the panel is still open, the area change is the verdict (3 s for the first column). Icons of rows 0..6 are U (never needed so far). Tests: `TestTakeWaypointIconColumn` (sim).
- Waypoint object (Lut Gholein class 156): `openWaypoint` still walks to `knownWaypointPos` (5069,5084, V) first; when `Interact` fails the hover check it now clicks the object plainly at (0,0), (-1,-1), (+1,+1) and accepts the panel as proof (like `hand.InteractNPC`, V for NPCs). Test `TestOpenWaypointPlainClickFallback`. U live.
- Which tomb: the game stores it. S: `docs/d2-1.14d-memory.md` 3.2 lists `ActMisc +0x94 dwStaffTombLevel (real tomb)` (D2BS D2Structs.h; prior-art/02-runs.md: Kolbot Duriel uses `getRoom().correcttomb`). U on 1.14d: nothing in the repo reads it. REQUEST for d2map/mem owner: `offMiscStaffTomb = 0x94` read next to `offMiscLevel1` in `Reader.Act` (`Act.StaffTomb uint32`, a level number 66..72 expected), expose it, and have the `a2env` in `tools/rigs/act2live` implement `quest.TombHinter` (`StaffTombLevel(ctx) (int, bool)`; false when the value is 0 or outside 66..72). Test with a world fixture that sets the dword. Live check: print the value in the Canyon before entering any tomb, then compare with the tomb that has class 152.
- Search (works without the hint): `quest.FindTomb` tries the hinted tomb first, then 66..72; a tomb is judged only after `TombProber.ProbeFar` ran (max 14 steps) and `HasObject(152)` is still false; a tomb whose gate fails (V live: exit 69) is skipped and retried after the others; `ErrNoTomb` carries the untouched tombs.
- `nav.FarProbe` (`internal/nav/probe.go`) is the real `ProbeFar`: targets = centres of the level's static Room2 list not yet visited (U: whether unstreamed rooms are listed), else 8 points on the level rectangle (inset 30); farthest first; `TeleportTo` with a key, else `ExploreToward` with a stop radius; a failed trip still counts as visited. Wire it in `a2env`: keep one `*nav.FarProbe` and `ProbeFar` = `Step`. U live.

## Tomb hint (TOMBHINT)

`Reader.StaffTombLevel(player)` and `act2live tombhint` (read-only) give the real tomb id; a2env implements quest.TombHinter and TombProber (one nav.FarProbe per rig). S offsets, U on 1.14d: a d2-live agent should run `tombhint` in the Canyon and compare with the tomb where `findtomb` finds object 152.

## Live results 6 (ACT2LIVE6, Navi on i1, 2026-10-29)

Tags: V live, U not seen.

| id | result |
|---|---|
| H1 | V: `Reader.StaffTombLevel` returns 69 both in Lut Gholein (area 40) and in the Canyon (`tombhint` -> `69 true`), so the offsets (player+0x1C pAct, Act+0x48 pMisc, ActMisc+0x94) give a value in 66..72. V: tomb 69 is the real tomb: `probefar` found object class 152 (orifice) in area 69 after 3 probe steps (teleport hops to far room centres), the other tombs (66, 67, 68) had none in their first rooms (ACT2LIVE5). So the hint and the search agree. |
| H2 | V: the exit gate of tomb 69 (12524,5285) is the mouth of a pillared cave at the cliff foot; the nav exit click/walk never fires there ("nav exit tile did not fire", stuck at (12525,5286)). What works: stand at (12530,5285) (`goto 12528 5285` after `clear`), `click 60 100` (moves to (12525,5285), in front of the pillars), then plain clicks at screen (340,175) and (300,190) (up-left, into the dark doorway): area 69 at (22873,9597). nav fix wanted (internal/nav owner): after the gate walk fails, click the screen point 60..100 px up-left of the player. |
| H3 | V: `TakeWaypoint(46)` from `wp 46` failed once ("level 46 not reached") although the panel was fine: with `obj 156` then `click 112 410` the Canyon loads. U why (3 s icon wait vs load time; the text-column fallback click may close the panel). |
| H4 | V (danger): between runs Navi dies within seconds in the Canyon and in tomb 69 (god off). Respawn: `key esc`, hold 4, `key esc`; life is then 0 (stat 6 absent, so `d2cheat heal` and `god` fail: "stat 6 not present") until one belt potion (`key 1`) is drunk; after that `heal` works. A whole leg must be one `act2live god N ... -- ...` chain. |
| P1 | `PlaceItem` now picks the item to the cursor (hand.Take), closes the panels and clicks the object (before: clicked the object without the staff). Results below. |

Open for the next session (restart i1 from the save: the Staff is back in the pack there): chain `act2live god 2400 wpgo 46 410 -- clear -- goto 12528 5285 -- click 60 100 -- hold 3 -- click 340 175 -- hold 3 -- click 300 190 -- hold 3 -- probefar 10 -- tpto 22519 9806 -- tpto 22519 9806 -- key i ... place hst 152`. Keep the Staff off the cursor across any run end; the orifice monsters (Burning Dead, Unraveler packs) stand on the platform and god does not stop them from killing Navi when the run ends.

## Offline fixes after ACT2LIVE6 (TOMBNAV, no live game)

- `nav.ExitTo`: for a tile exit, after the walk, the push, and the step-off retry all fail to change the level, `gateClicks` sends the plain screen clicks (60,100), (340,175), (300,190) (3 s area wait after each) before the gate is marked bad. Source: H2 (V live for tomb 69; U for other exits). Test `TestExitToGateClickFallback`.
- `nav.waitArea` (used by `TakeWaypoint`) extends its deadline while the snapshot reports a loading screen (up to LoadWait past the base wait), so the 3 s first wait no longer cuts the Canyon load. Whether that was really H3's cause is U. Test `TestTakeWaypointWaitsThroughLongLoad` (fails without the change). The text-column fallback click is unchanged.
- `PlaceItem` (tools/rigs/act2live/env.go) is unchanged: it finds the hover point first, takes the item to the cursor, closes panels, clicks.

### Next live chain for ACT2LIVE7 (U until run)
Staff is in the pack on i1 (if it is on the ground or cursor: `grab hst` then `key i` / put it in the pack first; never end a run with it on the cursor).
1. `act2live god 2400 wp 46 -- clear -- ...` (try nav `wp 46` first; fall back to `wpgo 46 410` if it still fails)
2. `-- exit 69` or the manual gate chain `goto 12528 5285 -- click 60 100 -- hold 3 -- click 340 175 -- hold 3 -- click 300 190 -- hold 3` (the nav `ExitTo(69)` should now do the clicks itself)
3. `-- probefar 10 -- tpto 22519 9806 -- tpto 22519 9806` (orifice platform; clear monsters first)
4. `-- place hst 152` (PlaceItem: GoToUnit, findHover, take hst, close panels, click). Check object 152 mode change and the Staff leaving the pack.
Whole leg in one god chain (H4: Navi dies between runs). Keep the cursor empty at the end, stand away from the platform.

# 03 Pathing and movement

Sources: KB:core/Pather.js (move 412-705, walkTo 776-905, openDoors 913-1000, moveToExit 1245-1335, useWaypoint 1503-1700, makePortal/usePortal 1704-1860, journeyTo 2030-2230, plotCourse 2231-2360), KB:core/CollMap.js, D2BS:Map/Path/Reduction/*, KO:internal/pather/*, KO:internal/action/step/move_to.go, TY:docs/features/pathing.md.

## 1. Two movement modes
| | walk (default) | teleport |
|---|---|---|
| path request | A* then "walk reducer", node spacing `walkDistance` = 5 (retry spacing rand 10..15) | A* then "teleport reducer", spacing `teleDistance` = 40 (retry rand 25..35; 30 in Maggot Lair) |
| step | click ground at node (run), wait until idle/arrived within `minDist` 4 (2 in town or after a failure) | cast Teleport at node, wait up to max(500ms, 2*ping+200) per attempt, 3 attempts, success when within 5 of node |
| use only if | always possible | not in town, not shapeshifted, skill available (or from items), mana allows > 2 casts (`mpmax/cost > 2`), and `tpMana <= mp` |
Fallback: when teleport mana is short the loop falls back to walking that node.

## 2. Path generation
- Host API `getPath(area, tx,ty, sx,sy, reductionType, spacing)`; reduction 0 = walk, 1 = teleport, 2 = "none/JS" (smooth). D2BS: A* on the merged collision grid; walk reducer takes straight segments while the line is walkable and within `range`; teleport reducer picks the farthest cell on a ring of radius `range` that is reachable and closest to the target (heuristic that halves search time). The A* in D2BS used in-process level map data (rooms) with unexplored rooms revealed on demand.
- Ours (`internal/nav`): own A* over `d2map` merged grid. KO's concrete scheme (MIT): 4-neighbor grid A*, cost 1 per plain cell, "soft blocker" cost 1000 for walkable cells within 2 cells of a wall (keeps path off corners), blocked cells excluded, Manhattan heuristic, optional extra padding, and destination cell neighborhood (3x3...7x7 for teleport) forced walkable to tolerate bad target cells. Path computed again whenever player strays more than 5 from the cached path (cache: keep path suffix from nearest node when nearest < 5).
- Collision flags used (KB sdk.collision): BlockWall 0x01, LineOfSight 0x02, Ranged 0x04, PlayerToWalk 0x08, DarkArea 0x10, Casting 0x20, Players 0x80, Monsters 0x100, Items 0x200, Objects 0x400, ClosedDoor 0x800, IsOnFloor 0x1000, FriendlyNPC 0x2000, DeadBodies 0x8000. Composites: BlockWalk 0x1805 (wall | ranged | floor-bit), BlockMissile 0x80E, WallOrRanged 0x5. In 1.14d the collision map word per cell comes from `CollMap` (memory doc 3.2); note flag meaning must be confirmed live (UNVERIFIED for our reader).
- `checkSpot(x,y,coll,size)`: a spot is valid if the 4 orthogonal neighbours at +-size (only cells with |dx| != |dy|) do not have the flag; default flag BlockWall; used by nearest-walkable search.
- `getNearestWalkable(x,y,range,step,coll)`: expanding square rings (distance 1,1+step..range) returns the first valid cell. Used for exits (range 5), maggot lair nodes (15, step 3), cow rooms (10, step 2). KO variant: range 20, step 4.
- `spotOnDistance(spot, dist)`: path to the spot, return the first path node closer than `dist` and valid; used for "stand N away from boss".
- `findSpotAtDistance`: try 16 angles at the given radius, then radius +-2.
- Level boundaries: KO merges neighbouring levels' grids when the destination is outside the current level grid (area transitions without loading screens, e.g. outdoor zones that share a level edge? In D2 each level is its own area; KO's `outsideBoundary` handles targets in an adjacent level region). UNVERIFIED relevance for 1.14d.

## 3. The move loop (KB `Pather.move`, behaviour)
```
move(target, settings{minDist 3, retry 5, allowTeleport, allowClearing, allowPicking, allowNodeActions, pop, callback}):
  if dead: fail
  close any open UI panels (inventory, stats, skills, npc menu, waypoint, party, shop, quest, trade, stash, cube)
  if item on cursor: drop it   (prevents mis-clicks)
  if within 2 and no missile-blocking collision: done
  if minDist > 3: retarget to a path node at that distance from the target
  path = getPath(...); reverse; optionally pop last node
  retry budget: if retry<=3 and target farther than 120 (tele) / 60 (walk) -> retry=10
  loop while nodes:
     node = next; callback() may end early (e.g. boss already killed)
     if dist(me,node) > 2:
        (maggot lair or every 3rd failure with invalid node) snap node to nearest walkable (15, step 3); small clear range 5; retry>=15
        ok = teleport? teleportTo(node) : walkTo(node, minDist = fail>0||town ? 2 : 4)
        if ok: optional node action (clear nearby monsters, open chests, shrines) once (guard against recursion), and if node far, re-path
        else (failed to reach node):
           if walking: try kicking barrels in the way or opening doors -> continue
           if allowed and monsters within 10 and (never cleared or cleared >3 s ago and far from last clear): Attack.clear(10)
           special skills: Leap Attack / Whirlwind toward the node (max once per 3 s unless mobs within 6)
           town special cases (Atma blocking Lut Gholein inn door (5136..5137, 5048..5051): take side nodes (5140,5038),(5148,5031),(5154,5025),(5161,5030))
           recompute path with smaller spacing; if fail>0 flash (re-sync the character) ; if fail >= retry: give up
           fail++
     5 ms delay
  success if final distance < 5
```
Numbers: `Pather.move` default retry 5, `minDist` 3, node reach 2, clear range 10, clear trigger 3 s, `walkTo` max 3 attempts with 3 stuck-fails (each 500 ms of "not moving" tries two perpendicular points 5 away).

## 4. walkTo (walk one node) details
- Run/walk switching to save stamina: if stamina < 20 percent drink a stamina potion if in inventory; switch to walk when <= 15 percent; back to run when >= recover threshold (80 percent if max stamina duration < 30 s else 50).
- Loop `while dist > minDist`: open doors near the destination (see 5); if attempt count > 1 and the straight line is blocked by a wall/closed door open doors near self; click the destination; wait up to 500 ms for `moving`; if never moving: stuck count++, click a point 5 away perpendicular (angle +-90 degrees from direction to target) that `validSpot`, wait up to 1 s; after 3 stuck results break. After 2+ attempts also kick barrels. Then wait until idle. Max 3 click attempts per call. Return dist <= minDist.
- Paladin Charge hop (>8 away, mana >= 9), Sorc Blaze (if mana > 2x cost and not on) while walking: class perks, optional.
- KO external equivalent (`move_to.go`): each tick: if teleport available select teleport skill (key) else Vigor; compute path (cached); walk step length = 25 x random walk duration (0.6..1.2 s) = 15..30 cells of path ahead, aim the mouse at path[len-N] converted to screen and press the bound "force move" key (D2R only). In 1.14d there is no such key: our `input` must click LMB on the ground at that screen point (Kolbot's `Misc.click` does exactly that, press then release 20 ms apart). Re-evaluate at most every 0.6..1.2 s walking, every `CastDuration` teleporting.
- KO timeout per MoveTo step: 30 s then mark complete (caller re-checks position). Completion when within 5.

## 5. Stuck detection and recovery
| bot | detector | recovery |
|---|---|---|
| KO | position unchanged for 20 consecutive samples (each sample taken on a MoveTo run) | blacklist the last path cell, re-run A*; if no path left, click a random point in the middle half of screen and force-move |
| KB walkTo | not moving 500 ms | perpendicular side-step 5 cells, 3 strikes |
| KB move | fail counter per path | repath with smaller spacing, `Packet.flash` (self-select packet to resync), snap node to walkable |
| TY | position/area progress watchdog; Navigator state stuck | abort navigation with reason `stuck` (fail closed) |
Edge cases: stuck at doors (A2 sewers, A5 gates), barrels/urns, narrow maps (Maggot Lair 1-3, Arcane Sanctuary, Claw Viper 2, River of Flame, Chaos) where teleport lands on invalid cells -> smaller range and nearest-walkable snapping; Arcane: platforms are non-floor, use floor bit + treat as soft blockers; dead bodies and items count as blockers for placement not for walking.

## 6. Doors, gates, obstacles (`openDoors`, `kickBarrels`)
- Door objects ("door" class, inactive mode). Door is relevant when within 4 of the path node and within 9 of player, or within 4 of player. Open: click it up to 3 times, wait <= 1 s each for mode Active.
- Act 5 gate (Harrogath <-> Bloody Foothills): same, 3 tries.
- Barricades (monsters named "barricaded door"/"barricade"): attack with primary skill up to 20 casts while hp > 0.
- Barrels etc.: names `ratnest, goo pile, barrel, basket, largeurn, jar1-3, urn, jug, barrel wilderness, cocoon` inactive and near the node: skip those with wall/ranged collision between; up to 5 interact tries (first 3 packet, then click) until mode changes.
- Town: doors only in Act 5.

## 7. Area transitions
- `moveToExit(area or [chain], use)`: find exits in `getArea().exits` with `target == next`; if several choose the nearest to the player; snap exit coordinates to nearest walkable (range 5); move there (3 retries, 200 ms apart, wait for gameReady); if `use` or not the final area: exit type 1 = walk-through (step into `getNearestRoom(target)` center), type 2 = stairs/object: first special cases `openExit` (A2 trapdoor, A3 sewer stairs, Arcane tomb stairs, Durance entry stairs, Ancients' door) else `useUnit(stairs, tile id)` choosing the nearest stairs unit. After arrival wait 300 ms.
- For us: `d2map` exits (memory doc 3.2 Level exits and warps) give target level id, position, and tile type; "walk through" = path onto the exit tile; stairs = click object/tile unit until `area id` changes (poll area every 50..100 ms, max ~3 s, retry 3).
- Wait for `gameReady`/loading: after any area change delay >= 500 ms (KB `useWaypoint` delays 500 after success "helps with crashes").

## 8. Waypoints
`useWaypoint(area)`: validates area in WP list (39 entries; Act1: Rogue, Cold Plains, Stony Field, Dark Wood, Black Marsh, Outer Cloister, Jail 1, Inner Cloister, Catacombs 2; Act2: Lut, Sewers 2, Dry Hills, Halls of Dead 2, Far Oasis, Lost City, Palace Cellar 1, Arcane, Canyon; Act3: Docktown, Spider Forest, Great Marsh, Flayer Jungle, Lower Kurast, Bazaar, Upper Kurast, Travincal, Durance 2; Act4: PF, City of the Damned, River of Flame; Act5 and others; classic has no Act5).
```
repeat up to 12 times:
  if in destination: done
  if in Lut Gholein and target in A1: talk to Warriv, "go west"; (A2 -> A1 boat)
  if in town and WP farther than 5 (20 with Telekinesis): walk to town spot "waypoint"
  find WP object (name waypoint) ; if not in town and farther than 7: moveToUnit
  if menu wanted/unknown: click WP; wait for WP UI flag (1000*(i+1)/(i/5+1) ms or 2*ping)
     if UI never opens: random move 5*retry away, flash, retry
  when UI open: if destination WP not yet activated -> cancel UI, fetch it by walking there (getWP), else
  click destination (wp.interact(area)); wait for area change; after: delay 1500; wait gameReady
  after 10 failures and not in town: take TP to town and use the town WP
```
Facts: WP-active bitmask cached after first UI open (`getWaypoint(index)`); you can never lose a WP, so cache positives. UI-open flag `Waypoint`. Our `internal/nav` needs: WP UI tab click coordinates per act and the area button (800x600 classic UI, calibrate live), `UIflags.Waypoint` from memory (2.11), active-WP bits (memory location UNVERIFIED; fallback: attempt click and check area change).
Getting a WP (`getWP`): move to preset waypoint object (by id list), click until UI opens (10 tries), then close. The first time in each area the bot touches the WP so all later runs can teleport.
Journey planning (`journeyTo`/`plotCourse`): a hand-written parent table `previousAreas[area]` (adjacency tree of all 136 areas) searched BFS from destination back to either the source or any active-WP area; result `course` plus `useWP`. Special hand-coded transitions: Stony Field -> Tristram (Rakanishu preset then portal), Lut -> Sewers (trapdoor), Palace -> Arcane (move (10073,8670) portal), Arcane -> Canyon (journal at preset, red portal), Canyon -> Duriel (tomb + staff holder), Trav -> Durance (preset stairs), Durance 3 -> PF (move (17581,8070) red portal), Harrogath -> Foothills (move (5026,5095) open gate), Harrogath -> Nihlathak (Anya's portal), Frigid/Arreat/Tundra red portals (Abaddon, Pit, Infernal), Cow/Uber portals via town stash. Precast before leaving each area; retry up to 4 times then fail.

## 9. Town portals (`makePortal`, `usePortal`)
- makePortal: need a TP tool (tome with quantity > 0 or a scroll). Remember the gid of own existing portal (portal "owner name" == player), use tool, then poll max(500+100*i, 2*ping+100) ms for a new portal object owned by me with a different gid; 5 attempts. Dummy portal object name bug: treat a "Dummy" portal as valid.
- usePortal: up to 10 tries: close UI; in town walk to "portalspot" after the first failure; get the portal (by target area and owner); if >5 away moveToUnit; enforce 2500 ms since last portal use (`lastPortalTick`) else wait; interact via packet (first 2) then mouse click; wait 500 ms for area change; flash every 2nd failure past 2. Special: Arcane portal needs click then wait 2 s for activation; red portals need +150 ms.
- `canTpToTown`: not dead, not in any of town/Arreat Summit/Uber Trist, has a tool.
- A town visit from the field (`goToTown`/`visitTown`): make portal, enter; do chores; walk to portal spot; take portal back (try own portal then area-matching portal); wait >= 4500 ms after arriving in town before returning (TownChicken) so monsters near the portal disperse? (it waits 4.5 s total from start).
- Town portal spot coordinates (world): A2 (5168,5060), A3 (5150,5063), A4 (5045,5042), A5 (5098,5019) (portal arrival) ; A1 derived: fire pit + (10,18).

## 10. Exploring unknown levels
- Kolbot never "explores": D2BS `getPath` and `getRoom` expose the full level map (generated on the client side) so it paths straight to exits and presets. Our reader (d2map) reads rooms only as the game loads them (Room2 list grows as player approaches). Therefore we need an explorer. Prior art: 
  - TY `ExplorePlanner`: bearing-first: rotating compass bearings (8) with step distance in tiles, bearing index resets on area change; success by area id.
  - KB `clearLevel`/`clearRoom`: iterate rooms from `getRoom()` sorted by distance, walk to each room center, clear 30, open chests.
  - d2mapapi (memory doc 3.3): generate the map from game data with a seed (offline: seed is in memory); recommended by memory doc as the long-term answer.
- Recommended order for `nav` explore (ours): (1) if exit/preset known, path to it; (2) else frontier = known-walkable cells adjacent to unknown cells; choose the frontier cell with best (distance, bias toward level direction). Rooms already loaded give the unknown boundary. UNVERIFIED: how far D2 loads rooms ahead (roughly 2 screens).
- Many levels are randomly laid out per game; exits and WP positions come from presets (preset units list per room, available once the room is initialised even if not yet seen? UNVERIFIED).

## 11. Recorded-route alternative (TY, D2R)
Idea: the human walks the route once; the bot replays it.
- Route contract v1 (TY): binding {character, class, difficulty, map seed, game version, profile id, layout fingerprint}; segments with world coordinates, area, movement kind, expected transitions. Recorder samples position when player moved enough, only from consistent in-game snapshots; area changes close a segment.
- Layout fingerprint = SHA-256 over sorted semantic anchors in the current area: waypoint, stash, portals, entrances (type id + world coordinates), NOT monsters/items/unit ids/player position. Playback refuses (fail closed) when fingerprint differs. D2R maps change per game, so a route is only valid when the same layout appears again (limited preset count: e.g. Act1 town has 4 exit variants; Tower Cellar levels vary).
- Playback: follow recorded world coordinates within a drift limit; local correction by navigator; transitions verified by area id; segment timeout is a progress watchdog reset by every confirmed new point; combat holds freeze the budget.
- Town routes are stored relative to the stash origin, keyed by fingerprint of the stash-to-waypoint vector (live finding: Act1 town layout changes per char/difficulty: 4 presets).
- Fit for us: levels in 1.14d are randomly generated by seed per game, so only fixed-layout areas (towns, Chaos Sanctuary seals variants A/B, Durance 3, Throne) are good for static data; for outdoor/dungeon areas A* + exit presets is the main path. Recording is useful as a fallback "teach the bot" tool for towns and the Chaos seals; use `d2map` presets as anchors.
- Honest limits: replay needs same layout; recorded paths bake in monsters' absence/presence; any drift needs navigator fallback.

## 12. Mapping
| need | package | data |
|---|---|---|
| A*, reducers, nearest walkable | `nav` | `d2map` merged walkability grid with flags (wall, door, object, floor) and per-room freshness |
| walkTo / teleportTo | `nav` + `input` | player pos (mem 2.3), moving/idle (player mode), skill ids and hotkeys (mem 2.8), mana, WorldToScreen |
| door/barrel/gate handling | `nav` | object list (name or txt id, mode) near path nodes |
| stuck detection | `nav` | position history (20 samples at tick rate), mouse/hover target (mem 2.12) |
| portals/WP | `nav` | object units: portal (owner name via parent unit id; mem 2.5), waypoint, area id, UI flags |
| exits | `nav` | `d2map` exits + tiles |
| explorer | `nav` | `d2map` room list, unknown mask |

# nav: walking, exits, waypoints, exploration (agent NAV)

## API summary (package `internal/nav`, `nav.New(e *env.Env) *Navigator`)

Implements the `Mover` contract of docs/archive/wave2.md. All coordinates are `input.Point` subtiles. One goroutine drives a Navigator;
nav is the caller of `Tracker.Update` (once per tick) and reads `d2.Reader.Snapshot` itself. Everything sleeps on `e.Clock`.

| method | what it does |
|---|---|
| `GoTo(ctx, p)` | plan on the current level, walk with clicks at `WaypointSpacing`, arrive within `ArriveTolerance` |
| `GoToUnitOf(ctx, typ, id)` / `GoToUnit(ctx, id)` | walk to within 4 subtiles of a monster/NPC (`d2.TypeMonster`), object (`d2.TypeObject`) or ground item, following a moving unit. Unit ids are only unique per type (the waypoint object and Warriv are both id 11 in town: VERIFIED LIVE, bug found that way), so use `GoToUnitOf`. `GoToUnit` searches monsters, then objects, then items. `ErrUnitGone` after 1 s missing |
| `ExitTo(ctx, levelNo)` | border gap (`LevelMap.BorderCells`) or `LevelMap.Exits` tile, then keep walking `ExitPush` past it until `Player.Area == levelNo`; explores when no way out is known yet; `ErrNoExit` when the level is exhausted |
| `TakeWaypoint(ctx, levelNo)` | walk to the waypoint (unit, else preset), `Interact`, wait for UI flag 0x14, click act tab + row (`wpPanel`), wait for the area change |
| `ToArea(ctx, levelNo)` | waypoint if the current level has one and the target is a waypoint level, else `Route` + `ExitTo` per hop; a hop that ends in `ErrNoExit` marks that edge bad and the route is recomputed (up to 4 times) |
| `Route(from, to) ([]int, error)` | levels to enter, in order. Edge sources by cost: exits seen in THIS game (tile exits and border gaps in the tracker's level maps, crossings made = 1), levels.txt `Vis` (2), static Act 1 hint graph `act1Hints` (3, FALLBACK, UNVERIFIED except 1-2, 2-3, 2-8). Edges are undirected once crossed. `ErrNoExit` across acts. The bot does not need its own graph any more |
| `ActivateWaypoint(ctx)` | find the waypoint of the current level (explores until its preset shows up), walk there, open the panel (this activates it), close it |
| `Explore(ctx, done)` / `ExploreToward(ctx, hint, done)` | frontier exploration until `done()`; `ErrExplored` when no un-inspected frontier is left |
| `PathTo(p)` | pure planning on the known grid (no unknown cells, no walking); `ErrNoPath` if unreachable |
| `CorpseApproach(ctx, corpse)` | GoTo with 2x monster avoidance; stops with `ErrDanger` when a living monster is within `CautionRadius` of the character or the corpse |
| `Guard func(*d2.Snapshot) error` | set by the caller; checked every tick, any error ends the walk as `ErrAborted` (wraps it) |

Errors: `ErrStuck, ErrNoPath, ErrUnitGone, ErrTimeout, ErrNoExit, ErrExplored, ErrNoWaypoint, ErrNotInGame, ErrAborted, ErrDead, ErrDanger`.
**Dead is a hard stop**: player mode 0 or 17 (docs/mem.md, UNVERIFIED live: no death happened during my runs; town mode is 5, neutral 1) or life 0 returns `ErrDead`
from every walk, before any click. Nothing in the ladder clicks a dead character.

Tuning (`internal/env/tuning_nav.go`, 20 knobs with ranges): PathQuality, Sloppiness, WaypointSpacing, ArriveTolerance, PollMs, ReplanMs, StuckMs,
MaxRecoveries, WallClearance, WallWeight, MonsterRadius, MonsterPenalty, ExploreInformed, ExploreBucket, WalkTimeoutSec, ExploreTimeout, ExitPush,
LoadWaitSec, PanelWaitMs, CautionRadius. Tests: path quality 1 > 0.5 > 0.1 gives longer planned paths and longer walked distance but always arrives;
spacing 4/8/24 gives fewer clicks, same arrival; poll 400 ms / stuck 4 s / replan 8 s still arrives.

## Design

- **Field** (`field.go`): dense copy of the grid window united with the level rectangle. Cell classes: walkable (known, BlockWalk clear), blocked, unknown
  inside the level (`fOpt`), nothing. Clearance by chamfer distance transform from known walls; wall penalty `(WallClearance - c) * WallWeight` per cell;
  object-flag cells (centre of stash/waypoint reads walkable, docs/map.md) cost extra; monsters add a fading disc (`MonsterRadius`, `MonsterPenalty`);
  blacklisted cells (stuck recovery) add heavy discs for 30 s. The field is rebuilt only when `KnownCount` changes (collision flag bits of moving things change the
  grid Version every tick, so Version is useless as a cache key).
- **Planner** (`astar.go`): 8-directional weighted A*, no corner cutting, generation-stamped scratch arrays, tie-breaking toward the goal. Unreachable goal:
  path to the explored cell nearest the goal (`partial`). Goal in unknown space or unreachable on known cells: second search with unknown-inside-level cells passable at
  +0.5 per step (free-space assumption, "optimistic"); the executor replans every ~500 ms as rooms load. This fixed the local-minimum trap of "walk to the closest point".
  Large fields (>= 180k cells): plain A* with a 40k-expansion budget, then a coarse corridor (5x5 blocks, +-2 blocks) and a fine search inside it, then unrestricted.
- **Post-processing** (`smooth.go`): string pulling by line of sight whose cost must not exceed the A* path cost between the same points (so shortcuts never pull a path
  back to a wall or through a monster); resampling at `WaypointSpacing` with every corner kept as a waypoint, each pair checked for line of sight. `PathQuality < 1`:
  A* heuristic weight `1 + 3(1-q)`, no smoothing below 0.5, sideways waypoint jitter up to `(1-q)*Sloppiness`.
- **Executor** (`walk.go`): tick = observe (wait through loading) -> guard/dead -> close panels -> replan if needed -> pick the farthest next waypoint with line of sight
  from the character (never cut a corner toward a waypoint behind a wall) -> click only when the target changed, the character stood still 2 ticks, or 3 s passed.
  Replan triggers: target moved, deviation > 2*spacing+4, `KnownCount` or nearby-monster set changed (not before `ReplanMs`), optimistic plan (every ~500 ms), partial plan
  reached its end. Stuck = no 2.5 subtile movement for `StuckMs`; ladder: re-click -> side step 5 -> (block the cells ahead, replan) -> open a door (objects.txt `IsDoor`, `Interact`;
  UNVERIFIED live) -> side step 10 -> random hop; from the second rung every rung also blacklists the two cells ahead. The ladder resets only after the character got
  12 subtiles from where it got stuck (a side step is not an escape). After `MaxRecoveries` rungs: `ErrStuck` (or success if within 1.5x tolerance of a unit target).
- **Other players** are not in the unit tables read here; if present as monsters/NPC units they are just penalised cells. Mercenaries are friendly monsters: same.
- **Explore** (`explore.go`): frontier = walkable cell with an unknown 4-neighbour inside `LevelMap.Rect` (cells outside the rectangle are other levels). BFS from the player,
  frontier cells grouped into `ExploreBucket` buckets; buckets the character stood in or already walked to are dead ends. Nearest mode: fewest hops. Informed mode: hops minus
  1.5 x min(cells,30) minus a heading/hint bonus. Each leg is a `walk` that stops as soon as the target stops being a frontier or `done()`.

## Verified live (the worker host, Rogue Encampment / Blood Moor, 2026-10-28, level-1 Sorceress "Testy")

- VERIFIED LIVE: town walking. GoTo waypoint (6044,4909), stash (6016,4934), Akara, Gheed, Charsi, Kashya: all arrive within 2-4.4 subtiles, 0.4-9 s per leg.
  Class ids as in docs/map.md (119 waypoint, 267 stash, 147/148/150/154/155 NPCs). Units far from the player are not in the unit tables (Akara at 60 subtiles): walk to a preset/known position first.
- VERIFIED LIVE: `ExitTo(2)` from town: gate at (5960,4952) (middle of the 15-cell gap y=4945..4959), push to (5952,4952), area becomes 2 in 2-8 s including the walk. `ExitTo(1)` back from the
  Blood Moor works the same way. Bug found live: from the Blood Moor side the contact strip also touches the town's wall cells, so the first gate guess (5959,4941) was a wall and the character
  got stuck; fixed by requiring the cell across the border in the neighbour's grid to be walkable, test `TestExitToThroughBorderGap`.
- VERIFIED LIVE: exploration in the Blood Moor loads more rooms as the character walks (24000 -> 43200 known cells in ~20 s). Runs ended early by the safety guard (monster
  class 5 at 35 subtiles, class 63): the retreat `ToArea(1)` brought the character back each time. No fight, no damage (life stayed 40/40).
- VERIFIED LIVE: waypoint panel opens on `Interact` in 150 ms; UI flag 0x14 (`UIWaypoint`) and 0x1 (inventory flag also set) read. Geometry from a screenshot (800x600): act-1 tab "I" at (115,77), rows at y = 138 + 35 row,
  names at x 160..300 (`wpPanel`). Esc (`CloseAll`) closes it. A new character shows only the act 1 tab and "No Other Waypoints Activated".
- (Superseded by the NAV2 section below: destinations and tile exits are VERIFIED LIVE.) Still UNVERIFIED: tabs of acts 2-5, doors, dead-state mode values, the whole corpse flow. No death occurred, so nothing about corpse position/respawn was recorded.
- Gotchas: player mode in town is 5 (not 1); town NPC units stay in the unit table after crossing into the Blood Moor (do not use "any monster near" as a danger test without filtering town NPCs);
  `Object`-flag cells are walkable on the grid but the footprint blocks (waypoint, stash): walking onto the unit cell gets stuck, so unit approaches stop within 4 subtiles and the final goal click is skipped for tolerance > 2.5.

## Performance (the head box box, Intel N95, Go 1.27, `go test ./internal/nav -bench .`)

Synthetic 1500x1500 level with 500 random rectangular obstacles, corner to corner (2100 subtiles):

| | time | expansions |
|---|---|---|
| build field (clearance etc.) | 45 ms | |
| plain A*, optimal | 240 ms | 649k |
| coarse corridor + fine A* (used when plain exceeds 40k expansions) | 50 ms | 70k |
| open field, plain A* | 3 ms | 1.5k |
| `TestPlanLargeLevelSpeed` end to end (field + plan + smooth + resample) | 114 ms first, 56 ms cached field | |
| Rogue Encampment (280x200) paths | about 2 ms | |

## Dead ends and decisions

- "Walk to the nearest reachable point when the goal is unknown/unreachable" gets trapped in local minima behind walls; replaced by optimistic planning through unknown cells.
- Using the tracker grid `Version` to detect new terrain: it changes every tick (player/monster flag bits); use `KnownCount`.
- Skipping waypoints by distance only ("within reach, take the next") cuts corners and pushes the character into walls; line of sight to the next waypoint is required.
- Resetting the stuck ladder on any 2.5 subtile movement let side steps reset it forever; now a real escape (12 subtiles) is required.
- Clearance-limited string pulling (endpoints near walls lowered the requirement) pulled paths back against walls; cost-limited pulling fixed it.

## Notes for other agents

- Use `GoToUnitOf(ctx, d2.TypeObject|TypeMonster, id)`; plain `GoToUnit` is ambiguous on id clashes.
- Set `n.Guard` for "life low / monster near": return an error to stop; `errors.Is(err, nav.ErrAborted)`.
- Treat `ErrDead` as the signal to hand over to the death/recovery logic; nav does no corpse pickup. Corpse run: `ToArea(deathLevel)` (waypoint from town, else chain), then `CorpseApproach(corpse)`.
- `nav` calls `World.Update()`; do not call it concurrently from another goroutine.
- Live tool: `tools/live.sh nav navlive <state|town|wpui [s]|out [s]|home|goto X Y|path X Y>` (`cmd/navlive` is a symlink to `internal/nav/cmd/navlive`).


## Seed independence, exits, waypoints (agent NAV2, instance i1 "Navi", 2026-10-28)

**No absolute world coordinates at runtime.** Every game has its own map seed: level origins differ and the town layout can even be
mirrored (Testy's Rogue Encampment: x 5960.., Blood Moor WEST of the town; Navi's: x 3840.., Blood Moor EAST). Everything derives from loaded level
data (`LevelMap.Presets`, `BorderCells`, `Exits`, unit tables). Done: `navlive town` and `navlive out` (preset positions, town rectangle
from the tracker) and `internal/town` (the `Act.Approx` coordinate tables are gone; `Town.spotOfClass/spotOfObject` read the preset of the current level,
so the fallback walk when an NPC is not loaded yet is right in every seed). Tests run on two recorded games: `internal/d2map/testdata/rogue_encampment.d2mi`
(Testy) and `rogue_encampment_b.d2mi` (Navi, recorded with `d2mapdump record`); `TestRogueEncampmentPaths` / `...GateIsABorderGap` derive all targets from
presets. Remaining literals: `internal/combat/cmd/combatlive` (x 5935..6005, y 4952; COMBAT's tool, must go), `internal/town/sim` and unit tests use made-up
coordinates in fake worlds (harmless, not live), and d2map's fixture test asserts the recorded values of fixture A on purpose.
`navlive town` on Navi's game: waypoint, stash, Akara, Gheed, Charsi, Kashya all reached within 1.4-5.7 subtiles (VERIFIED LIVE).

**Exits (VERIFIED LIVE on Navi's seed):**
- Rogue Encampment -> Blood Moor: border gap (east here, west on Testy's seed), 13 s including the walk.
- Blood Moor -> Den of Evil: a TILE exit. Preset type 5 (tile id 2) plus `RoomTile.Dest` -> `Exit{Dest:8}` shows up in `LevelMap.Exits` as soon as the Room2
  holding the preset is loaded (the Den entrance appeared while the Blood Moor was explored). `ExitTo(8)` walks onto it: level changes in about 7 s.
- Den -> Blood Moor: the return tile (preset id 4) sits exactly where the character arrives. `ExitTo(2)` worked with the plain click-on-the-spot, and there is a
  fallback (`stepOff`: walk 6-12 subtiles away, then back onto the tile) for when standing on it does nothing.
- Blood Moor -> Cold Plains and back: border gaps. Full circuit Town -> Blood Moor -> Den -> Blood Moor -> Town via `ToArea(8)` / `ToArea(1)` (46 s of exploring for the
  Den entrance on a fresh tracker, 14-80 s total).
- Bugs found and fixed: (1) a tile fires before the walk reaches its middle and the walk then kept heading for coordinates of the OLD level: `walk` returns
  `ErrLevelChanged` now and `ExitTo` treats it as success when the new level is the target. (2) Exploring for an exit stopped as soon as the tile preset existed
  although no path to it was known yet; the stop condition is now "a gate exists AND is reachable over known cells" (`gateReachable`). (3) From the Blood Moor
  side the border strip toward the town includes wall cells the town grid did not show (the town was never loaded in this process): failed gates are blacklisted
  per (level, target, 6 subtiles) and the next gap is tried (8 attempts); the blacklist clears on success. (4) Duplicate `Exits` entries from overlapping rooms.
- Doors: nothing met on this route (UNVERIFIED live; the stuck ladder still has the object `IsDoor` rung).
- Dying: a dead character in the Den (god mode is only active while a navlive run is alive; between runs the level-12 sorceress with 51 life was killed while idle). `navlive respawn`
  presses Esc until alive: respawn in the Rogue Encampment VERIFIED LIVE (mode 17 read as dead, nav returned `ErrDead` and clicked nothing).

**Waypoints (VERIFIED LIVE):**
- Touching a waypoint (`ActivateWaypoint`: GoToUnit, Interact, panel opens in 150 ms, Esc) activates it. In Cold Plains it took 6 s including finding it.
- Panel geometry from a screenshot (act 1): ALL nine act-1 slots are drawn at fixed rows whether active or not (inactive ones grey, active blue); row text y = 135, 169, 203, 238, 272, 306, 340, 375, 409
  (pitch 34.25; constants RowY0 135, RowDY 34), x about 160..300, act tab "I" at (115,77). Order: Rogue Encampment, Cold Plains, Stony Field, Dark Wood, Black Marsh, Outer Cloister, Jail 1, Inner Cloister, Catacombs 2.
- Rogue Encampment -> Cold Plains and Cold Plains -> Rogue Encampment through the panel (act tab click, row click, area change): VERIFIED LIVE, 1.8 s each when standing at the waypoint.
- UNVERIFIED: tabs of acts 2-5 (x extrapolated at the pitch 57: 173, 230, 287, 344), their row lists (wpLevels order from levels.txt knowledge), clicking an inactive row (should do nothing: `ErrNoWaypoint`).

**Knob keys:** the `nav.exploreTimeoutSec` key matches `ExploreTimeoutSec`. The remaining mismatch was `loot.minValuePerCell` (field typo `MinValuePerCel`, fixed by renaming);
`go test ./internal/env ./internal/bot -run EveryKnob -v` has no mismatch line now.

**Notes for the bot/main agent:** drop the bot's own fallback graph, call `ToArea` (it routes itself and learns). `ErrLevelChanged` can come out of any `GoTo` that crosses a tile. Do not leave the character
idle outside town; god mode exists only inside a running `navlive --god` process. `navlive` flags: `--debug` (verbose), `--god` (logged cheat, actor nav2), commands `exits explore exit toarea wp wpactivate respawn`.

## NAVFIX (instance i1, 2026-10-28): "no exit to that level" right after arrival

Soak symptom (LIVEBOT, i4): ~8 failsafe rungs per hour, `ExitTo` says `ErrNoExit` within 50 ms of "no exit known yet, exploring" and `Explore` returns `ErrExplored` at once
(Blood Moor to Cold Plains, Den to Blood Moor, Stony Field to Dark Wood).

**Root cause (VERIFIED LIVE on i1, `navlive circuit 2 8 2 3 4 3 2 1 2 1`, before/after):** `Tracker.scanStatic` REPLACED a level's exits, adjacency and per-room presets with whatever the
currently loaded Room2s listed. The client frees a Room2's preset/tile/near data when the player walks away (and gives Room2s new addresses when a level is entered again), so on every
revisit the level forgot what it knew: Blood Moor from Cold Plains showed `exits=[]` (Den entrance gone) and no town border (`adj` lost level 1); the town showed 466 presets (233 twice,
old and new Room2 addresses). The grid itself is kept, so an already explored level has no frontier: nav found no gate, `Explore` answered `ErrExplored` immediately (the bot read that as
"area cleared"), `ExitTo` turned it into `ErrNoExit`. Before the fix BM to town took 45 s of re-exploring; after, `nav exit` directly.
Fix (`internal/d2map/tracker.go`, `static.go`): presets, exits and adjacency now only accumulate (presets de-duplicated by kind/id/position); `Level.Rect` is refreshed if the first read was empty.
Nav: `ExitTo` waits up to 3 s for the map to grow once before giving up with `ErrNoExit` (rooms still loading right after an arrival).
Tests: `TestStaticDataSurvivesRoomDataBeingFreed`, `TestMergePresetsDropsDuplicates`, `TestLevelRectFilledInLate`.

**Proof:** 10 consecutive transitions Rogue Encampment, Blood Moor, Den, Blood Moor, Cold Plains, Stony Field, Cold Plains, Blood Moor, town, Blood Moor, town: all 10 arrived, zero `ErrNoExit`.
The only `ErrExplored` seen is the fully known town (legitimate).

**Tracker.Reset review (LIVEBOT hypothesis "stale address caches after menu re-entry"):** NOT proven. `navlive reentry` (visit BM, Save and Exit, re-enter, ExitTo(2)/(1) with the same tracker, no Reset) worked.
Facts it did show: the act seed and level rectangles are the same after re-entry (same map), Room2 addresses change (presets doubled). Reset stays (harmless, drops address-keyed state), but it was not the
cause of the soak failures; the failures also happened without any re-entry (Burial Grounds, Dark Wood).
`navlive circuit L1 L2 ...` and `navlive reentry [reset]` are the repro tools.

## NAVFIX2 (instance i3 "Towni", level 20 throwaway, 2026-10-28): full Act 1 graph walk

Method: `navlive --god chain|chainwp L1 L2 ...` (ToArea per level, fresh tracker per run, one line per hop; `chainwp` also touches each waypoint; `xN` = `Explore` until the exit to N is known, then take it),
`navlive route A B` (planned route plus levels.txt Vis), `navlive exits` (all known levels, adjacency, tile presets), `navlive tiles`. Cheats (ledger, actor navfix2): level 20, movement speed 200
(without it every dense pack blocks clicks), `waypoints all` did NOT show in the panel afterwards (U, CHEAT's area). Seed of this game: town x 5600.., Blood Moor south, Cold Plains west of it.
Tags: V = verified live this session, S = seen in data/logs only, U = not reached.

| transition | exit is | ExitTo/ToArea | time | notes |
|---|---|---|---|---|
| Rogue Encampment 1 -> Blood Moor 2 | border gap (15 cells), push 8 subtiles | ok V | 11-14 s | back 2 -> 1: 42 s (walk across the Blood Moor) |
| Blood Moor 2 -> Den of Evil 8 | tile exit (preset 5, `Exits` Dest 8) | ok V | 29-59 s (explore until the room with the tile loads) | tile found as the Room2 loads, not before |
| Den 8 -> Blood Moor 2 | tile exit at the arrival spot | ok V | 1.3-3.7 s | |
| Blood Moor 2 -> Cold Plains 3 | border gap | ok V | 14-20 s known, 60-80 s fresh | |
| Cold Plains 3 -> Cave level 1 (id 9) | tile exit (Dest 9) | ok V | 8-11 s | levels.txt calls ids 9/13 "Cave 2"/"Cave 2 Treasure" (area id vs name mismatch, below) |
| Cave 1 (9) -> Cave 2 (13), stairs down | tile exit (Dest 13) at the far end | ok V | Explore to the exit 86 s, walk 1m44 total | `Explore` returns nil once the exit is known; stairs ARE tile exits, no object |
| Cave 2 (13) -> Cave 1 | tile at the arrival spot | ok V | 1.3 s | Cave 2 is a dead end: `Explore` right after arrival says ErrExplored (legitimate) |
| Cave 1 -> Cold Plains | tile at the arrival spot | ok V | 1.3 s when standing at it, 67-76 s from the stairs | |
| Cold Plains 3 -> Burial Grounds 17 | border gap (not a tile) | ok V | 12-88 s | |
| Burial Grounds 17 -> Crypt 18 / Mausoleum 19 | tile exits (Dest 18, 19) | ok V | 12-34 s | back: tile at arrival, 0.9 s |
| Cold Plains 3 -> Stony Field 4 | border gap | ok V | 23-100 s | |
| Stony Field 4 -> Underground Passage 1 (id 10, "Cave 3") | tile exit | ok V | 99-166 s (a monster pack on the cairn stones slows the walk) | |
| Underground Passage 1 (10) -> Dark Wood 5 | tile exit (Dest 5) | ok V | 82-119 s | the only way on foot; no Stony -> Dark Wood link |
| Dark Wood 5 -> Black Marsh 6 | border gap | ok V | 25-29 s | |
| Black Marsh 6 -> Forgotten Tower 20 ("Tower 2") | tile exit | ok V | 9 s, back 1.2 s | the tower hangs off the Black Marsh, NOT the Tamoe Highland |
| Black Marsh 6 -> Hole 1 (11) | tile exit | ok V | 72 s (the walk stood 2 subtiles from the tile for ~40 s before it fired, U cause: load) | back 1.2 s |
| Black Marsh 6 -> Tamoe Highland 7 | border gap | ok V | 40-65 s | |
| Tamoe Highland 7 -> Pit 1 (12) | tile exit (preset id 3) | ok V | 38 s | Pit 2 (16): exit found, walk blocked by a pack (ErrStuck, not ErrNoExit) |
| Tamoe Highland 7 -> Monastery Gate 26 | border gap (rect strip 320 wide, contact 1 subtile) | ok V | 28-74 s | |
| Monastery Gate 26 -> Outer Cloister 27 | border gap | ok V once (8 s), ErrStuck twice | | the gate's guard pack blocks the click path (level 26 position stays put) |
| Outer Cloister 27 -> Barracks 28 | border gap | ok V (21 s) | | |
| Barracks 28 -> Jail 1 ... Inner Cloister, Cathedral, Catacombs 1-4 | S/U | U | | blocked by monster packs in the Barracks corridor with a character that cannot fight (nav does not fight): not walked |
| Waypoints | object 119 etc., panel | ActivateWaypoint V | 7-61 s incl. exploring for it | TakeWaypoint town -> Black Marsh V (see NAV2); rows of Catacombs 2 U |

Proof run (final binary, one process per chain, `navlive --god chain 2 8 2 3 x9 x13 9 3 17 18 17 19 17 3 2 1 2 8 2 3`): 20 consecutive transitions through Den of Evil, Cave 1 and 2, Burial Grounds,
Crypt, Mausoleum, zero `ErrNoExit`, zero errors, including `Explore` of Cave level 1 until its exit to level 13 was known (86 s, returned nil) and the stairs taken.

### Failures found and fixes (offline tests in internal/nav/exit_test.go)

The campaign's `ErrNoExit` (Den -> Blood Moor, Cold Plains -> Cave 1) did not reproduce on this build: both exits are plain tile presets and worked every time (the NAVFIX accumulation fix is in).
Other causes of "no exit", all found live and fixed:
1. **A walk that ends in a third level.** From the Monastery Gate a border push fell back into the Tamoe Highland; `ExitTo` went on exploring THAT level for an exit to 27, returned `ErrNoExit`, and `ToArea`
   marked the edge 26 -> 27 bad for the game: every later hop (28 .. 37) failed with "no route". Now `ExitTo` returns `ErrLevelChanged` (wrapping the levels) when it ends in a level that is neither start nor target,
   `ToArea` re-plans from where it is and only marks an edge bad if the level did not change. `TestExitToEndingInThirdLevelIsNotNoExit`.
2. **Wrong hint graph.** `act1Hints` had Stony Field - Dark Wood, Dark Wood - Underground Passage 2, Tamoe Highland - Forgotten Tower; levels.txt Vis says: Stony 4 - UP1 10 - Dark Wood 5 (and 10 - 14), Black Marsh 6 - Tower 20,
   Tamoe 7 - Pit 12 - 16. A false direct edge is cheaper (3) than the real two-hop route (6), so `ToArea(5)` from Stony would explore the whole Stony Field first. Fixed, `TestRouteHintsFollowRealLayout`.
3. **Waypoint chosen for a one-hop trip.** `ToArea` walked to (and explored for) the waypoint even to a neighbour: 60 s in Cold Plains for a waypoint that was not even activated. Waypoint now only when the route has more than one hop.
4. **Walking onto the waypoint preset.** Before the unit appears `openWaypoint` walked to the preset itself; the object's footprint blocks, the stuck ladder clicked the waypoint (panel opened) and the next walk click hit
   the panel's first row, sending the character to the Rogue Encampment. It now stops 5 subtiles short (`TestOpenWaypointStopsNextToPreset`) and approaches from another side when the pointer never hovers it (`TestOpenWaypointRetriesFromAnotherSide`).
5. Not nav: with a character that does not fight, packs (Stony Field cairns, Pit, Monastery Gate, Barracks) block the click path; the stuck ladder ends in `ErrStuck` after about 15 s. Movement speed x2 only helps in the open.
   A killed `--god` process leaves the inflated max life (8191) behind (CHEAT's note in docs/cheats.md); `d2cheat level N` resets it.

### How exits are represented (V)
- Town <-> wilderness, Blood Moor <-> Cold Plains, Cold Plains <-> Burial Grounds / Stony Field, Dark Wood <-> Black Marsh <-> Tamoe Highland <-> Monastery Gate <-> Outer Cloister <-> Barracks: **border gaps** (`Adjacent` contact strip, `BorderCells`).
- Den, Caves, Crypt, Mausoleum, Underground Passage, Tower, Hole, Pit: **tile exits**: preset kind 5 plus `RoomTile.Dest` gives `Exits{Dest}`. The way back is a tile exactly where the character arrives; `ExitTo` clicks it from the spot.
  Stairs down inside a cave are the same kind: no "next level" object exists to look for.
- Area id vs level id: levels.txt names do not match the game's names (id 8 "Cave 1" = Den of Evil, 9 "Cave 2" = Cave level 1, 13 "Cave 2 Treasure" = Cave level 2, 10 "Cave 3" = Underground Passage 1, 11 = Hole 1, 12 = Pit 1, 20 "Tower 2" = Forgotten Tower).
  `Player.Area` and `Exit.Dest` use the same ids, so nothing is mismatched in nav; only reading level names from levels.txt misleads.
- `level.Rect` of caves is a placeholder (1000x1000 for Den and Cave 1, 120x120 for Cave 2): do not use it as the level size.

## Exit-tile avoidance and pack stalls (PORTALNAV, offline)

- **Exit tiles (V in tests, live effect U):** `plan` adds a cost disc (radius 6, peak 40, `exitAvoidR/Peak` in nav.go) around every `LevelMap.Exits` entry that is not the walk's own target: exits to `Navigator.wantExit` (set by `ExitTo` for its destination) and exits within 8 subtiles of the goal are skipped. It is a cost, not a wall, so a corridor whose only way is over an exit still plans (test `TestPlanThroughExitWhenOnlyWay`). Cause of the live Maggot Lair entry (V from docs/act2.md nav2): the straight path to the Far Oasis waypoint crossed the lair's tile. Only preset tile exits are covered; border gates (outdoor level edges) are not penalized (U). The exit tile size and whether `Exit.Pos` is its centre are U.
- **Pack stalls:** when the stuck timer fires with a living non-NPC monster within 8 subtiles, `Navigator.Yield(ctx, near)` is called (at most 4 times per walk); the bot should install a combat clear there. nil error: nav replans from rung 0; error: the walk ends with it. Without `Yield`, rungs 1 and 4 of the recovery ladder retreat from the pack centre (10, 6, 3 subtiles away) and blacklist its cells instead of a blind side step. Tests: `TestPackStallYieldsToCombat`, `TestPackStallWithoutYieldRetreatsThenFails` (synthetic corridor with solid monsters).
- **Not fixed (U):** the Halls of the Dead 3 chamber behind a wall the walk grid does not path through still ends in `ErrStuck` (nothing known leads there; docs/act2.md F1 used Teleport). Needs a live map dump of that chamber to tell a missing room from a door.

## Teleport exploring (teleport.go)

Set `Navigator.TeleportKey` ("f2" in the Sorceress plan; optional `TeleportRange` 18, `TeleportMana` 24, `Refill` hook). Then:
- `Explore`/`ExploreToward` pick the best frontier bucket on the whole known grid (reachable by walking or not; score = distance - 0.5*size, minus a hint bonus) and hop toward it: the landing cell is the walkable known cell within 18 subtiles that gets closest to the frontier. Walls in between are ignored (V live, docs/act2.md: Teleport crossed the Halls 3 wall; about 18 subtiles per cast).
- `ExitTo` hops to within 10 subtiles of a known tile gate before walking the rest (border exits are walked as before).
- Each hop is checked by position; cells where nothing happened are not retried. Three non-moving hops in a row, no usable cell or no mana within 40 s make that leg walk (the old behaviour). Mana below `TeleportMana` is waited out (or `Refill` is called); MaxMana 0 counts as unknown and casts anyway.
- Status: fake-map tests only (U live). The Arcane Sanctuary run is the live check: `ExploreToward(summoner preset)` with the key set; watch whether casts land and whether the maze grid keeps streaming in. Teleport is forbidden in Duriel's lair (levels.txt Teleport=2), do not set the key there.

## NPC interact fallback (internal/town/hand/interact.go)

`Hand.InteractNPC` = `Interact` (hover verified); on `ErrNotHovered`/`ErrNoHover` it clicks the unit's world position (then -1,-1 and +1,+1) and accepts the click when the NPC menu opens (U live: the hover failure is from Live results 3, NPC on stairs/off-centre). `town.openNPC` uses it.

## Island hopping (teleport_islands.go, ARCANE)

`Navigator.TeleportTo(ctx, x, y)` / `TeleportToStop(ctx, x, y, stop)`: needs `TeleportKey`; returns `ErrNoPath` when hops stop working or make no progress (8 hops without getting 2 subtiles closer, 12 dead blind casts, 80 hops). Callers walk the rest.
- Planner (`planHops`): A* over known walkable cells (every 4th cell plus every island rim cell), edge = within cast range (18) and at least 4 apart, void or walls in between do not matter, cost 1 per hop. Re-planned after every hop because rooms stream in. If no known cell is near the target it leads to the reachable cell nearest the target.
- Blind step (`blindCell`): when the plan cannot get closer, cast at an UNKNOWN cell within range that is at least 2 nearer the target, preferring cells inside a static Room2 rectangle (`LevelMap.Rooms`, a level room that has not streamed in). Each cast is verified by position; a cast that does nothing blacklists a 7x7 block.
- `teleLeg` (Explore/ExploreToward) falls back to this when the greedy hop finds no known cell (the Y3 stall at 4 hops), aiming at the hint or the chosen frontier.
- Tags: fake-map tests only (islands over a void, streaming rooms): U live. U: that unloaded Arcane rooms appear in `LevelMap.Rooms` before the player is near (the static Room2 list is read by the tracker, but whether the whole level is listed is unverified), U: blind casts into a never-loaded room land (Y4 saw blind casts stop at the first void edge, which this treats as partial progress), U: cast range above 18.
- Offline derivation of the layout was not possible: no DS1/room data is in the repo, only what the tracker reads from memory.
- boss: `Anchor.Teleport` plus optional `boss.Teleporter` on the Actions value; `summoner()` sets it. Adapters in cmd/bossrun and cmd/d2bot need `func (a bossActs) TeleportTo(ctx, p boss.Pt) error { return a.g.nv.TeleportTo(ctx, p.X, p.Y) }` (not done: outside my directories).

## NAVSTALL (offline, 2026-10-29): the 65 s stall leaving Stony Field, and waypoint activation in passing

Source: `livebot/20261029-053830-d2bot.log` on the worker host, 06:36-06:41 (Booti i4, level slice `level.14`). Tags: V = read from that log or reproduced offline, U = not seen live.

**Cause (V offline, reproduced by `TestLevelToTravelsWithoutGuardTarget`, which fails without the fix; the live confirmation needs the game, U).** nav was never at fault: the log has no `nav exit` line between arrival and the failsafe, and the
hint graph is right (Stony Field 4 -> Underground Passage 10 -> Dark Wood 5, `TestRouteHintsFollowRealLayout`). `LevelTo` calls `target()`, which publishes `ob.targetArea` for the guard; the task layer runs the slice in state
`clearing`, and the guard cancels a clearing action whenever the player is outside `targetArea` outside town ("exploring crossed a border: walk back", silent `errReplan`). The walk to the target is exactly that. So `travelTo` started, tried
the static waypoint list first (Black Marsh then Dark Wood, ~22 s each because `nav.TakeWaypoint` ignores ctx while it waits for the click to do something), then `ExitTo` ran on a cancelled context and returned
at once. The next slice repeated it: no movement, no log, "no progress" after 60 s, failsafe ladder. The same cancel hit every field-to-field slice; slices that happened to start in town (the guard exempts it) worked.
Fix: `taskActs.LevelTo` clears `targetArea` while it walks (`setTargetArea`) and publishes it again for the sweep. `Clear` already did the same.

**Waypoint trips that cannot work (V from the same log: "level 6 not reached (waypoint not activated?)" about 20 s each, then again next slice).** `travelTo` used `Plan.Waypoints`, the static list of every waypoint. Now
`activeWaypoints` (save/memory reader through `TaskDeps.Quests`, `taskState.wp`, plus what this runner touched) says which are active; with a reader a waypoint known inactive is not tried, without one it is tried once and a failure is not repeated for 10 minutes
(`wpState.failed`). A successful waypoint trip proves the origin and destination active and records both. `nav.waitArea` now honours ctx.

**Activating in passing.** `nav.ActivateWaypointNear(ctx, maxDist)`: touch the current level's waypoint if it is already known (live unit or preset) within `maxDist` subtiles, never explore. The bot calls it (`touchWaypointHere`, via the optional
`bot.WaypointToucher` interface that `moverAdapter` gets for free from the embedded Navigator) at the start of every field sweep pass and before every walk hop, at most every 20 s per area, only with nothing hostile near, only when the area has a waypoint that
is not known active, 3 failures per area and difficulty at most, max distance 70 subtiles. A touched area goes into the World's `WPs`, so `unreachable()` accepts routes through it (Cold Plains / Stony Field then make Black Marsh and later areas reachable)
and `travelTo` uses it. Town waypoints are not handled here (the campaign has `StepWaypoint`). Tests: `TestActivateWaypointNear`, `internal/bot/waypoints_test.go`.

Not verified live (U): that the waypoint unit is in the unit table or presets at the 70-subtile range on arrival (Cold Plains and Stony Field waypoints are in the middle of the map; the preset is known from the map server, the unit streams in nearby), that the
walk to it costs only seconds, and that the touched state survives in the save for later games (the runner remembers it per runner, not per game).

## NAVSTUCK: the same stall twice (Burial Grounds, 2026-10-30, offline fix)

Symptom (soak24 G1 logs on the worker host, V): `exit to Cold Plains: nav: stuck at {5459 4549} going to {5421 4599} (after 6 recoveries)` at 00:37:28 and again at 03:33:46 / 03:35:05 / 03:37:22, then the
failsafe skipped the area. NOTE: the area is **Burial Grounds (17)**, not Stony Field: the planner sent the bot Burial Grounds -> Cold Plains -> Stony Field (V, `nav exit to:3` right after
`Entered Burial Grounds`). The first stall (00:36:19, going north to a clear target) was at {5459 4552} wp {5459 4548}, the later ones at {5459 4549} wp {5459 4552}: the character does not move at
all for minutes (status lines show the same x,y with `hostileNear 1`), through re-click, side steps, hop and two combat yields. Position and waypoints are identical in both hours, so it is static
geometry or a static object, not a pack.

Root cause: NOT proven. No map dump of area 17 or debug click trail exists in the logs; the soak runs at info level. Hypotheses (U), in order of likelihood: (a) a thin solid cell or object
footprint on the corridor at {5459 4550} that the Grid reports walkable (the planner then plans straight through it again after every replan, because the old blacklist was only a cost disc of 30
around two cells ahead); (b) a click that lands on a monster or object near the waypoint (the character attacks or opens it instead of walking); (c) a monster stuck in geometry (`hostileNear 1` stays 1).
To settle it: replay with `NAVLOG=1` equivalent at debug (`nav click`, `nav plan` lines) or `d2mapdump` of area 17 around (5459,4550) on that seed.

Product fix (internal/nav/stuckmem.go; tests `stuckmem_test.go`, all offline in the sim, V):
- **Stuck-spot memory:** each stall is recorded (level, position, time, walk number). A stall within 10 subtiles of one from an EARLIER walk in the last 10 min logs `nav: stuck again at the same spot`
  and uses the alternative ladder for rungs 0-1: `nav: blacklisted cell` (a HARD block, `Field.AddBlock`, radius 2.5 + 1.5 per hit up to 7, centred 4 subtiles ahead, 3 min) then a spiral side step
  (cells 55..125 degrees off the heading, 4..14 subtiles, alternating sides, never one already tried at this spot). Rungs 2.. are the old ladder. Hard blocks are skipped near the start and goal, and if
  they leave no path `plan` retries without them.
- **Exit approach:** when `ExitTo`'s walk ends in `ErrStuck`, it marks that gate cell bad for 2 min (`stalledGateTTL`; ordinary bad gates stay for the game) and walks to up to two other cells of the
  same border gap (5..16 subtiles away), `nav exit: other approach cell`.
- **Escape hook:** if that fails and `Navigator.Escape` is set (d2bot installs `town.ToTown`, a town portal), it is called once per `ExitTo`; the call then returns `ErrLevelChanged` and the task/route
  re-plans from town, where `nav.useWaypoints` gives Cold Plains by waypoint. Without the hook the original `ErrStuck` is returned and the failsafe skip (navSkipSec 150 s) applies as before.
- **Logs:** `nav: stuck recovered at=.. method=reclick|sidestep|replan|door|wide-sidestep|hop|spiral rung=.. repeat=..` once the character is 4 subtiles from where it stalled.
- Not changed: ladder length, the 30 s cost blacklist, the failsafe skip. Unverified live: whether any of this frees the real spot (needs the cause above). If the character is truly rooted (clicks do
  nothing), only the Escape hook helps.

## NAVMEM: the 28 GB runaway (G1 attempt 16, 2026-10-30; offline fix)

Evidence (V): at 04:42:40, in the fourth `nav stalled by monsters, yielding` of one walk in Burial Grounds (right after
the town portal, `level changed during the walk: 1 -> 17`), the log went silent. The next d2soak sample (04:42:55) read
heap in use 27.8 GB, RSS 32.5 GB, 31 goroutines (the sample 30 s earlier: 43 MB). The OOM killer ended the process at
48.8 GB. One goroutine, no logging, live memory growing at about 1 GB/s: a single call allocating without a bound.

What the yield path calls (V, read): `combat.Clear` -> `awayPoint` -> `Fighter.valid` -> `Navigator.PathTo` (up to 30
plans per call) -> `buildField` + `Planner.search`. Both sized themselves from `lm.Rect`, the level rectangle of the
Level struct. The reader accepted up to 4096 tiles per side (20480 subtiles, 420 M cells; 13 bytes per cell for the
field and the planner scratch is 5.4 GB per plan), and the grid cap (16 M cells) did not apply to the union with that
rectangle. The cells outside the known grid are `fOpt` (optimistically walkable), so an unreachable candidate point
made the unbounded final `search` flood all of them. U: that this attempt's Level struct was torn or uninitialised
(it was read at the moment of the area change); it is the only unbounded structure on the path and it matches the
numbers, but no heap profile exists to confirm it. The heap guard (docs/stallfatal.md 7b) writes one next time.

Fixes (V offline, `internal/nav/memcap_test.go`):
- `buildField` clamps the window to `maxFieldCells` (4096 x 4096, the Grid cap): an absurd level rectangle is cut to a
  400 subtile margin around the known area, or dropped; nothing known and a huge rectangle gives an empty field
  (`ErrNoPath`).
- `Planner.search` stops after `maxExpandAbs` (= maxFieldCells) expansions and returns its best partial path
  (`Planner.Truncated`); `trace` cannot loop on a parent cycle.
- `d2map.Reader.Level` rejects a level larger than 600 tiles per side (`maxLevelTiles`, S: real levels are under 400).

Before/after, `TestPlanBogusLevelRectIsBounded` (a 60 x 40 known map in a 6000 x 6000 level rectangle, goal in a closed
pocket): before 36 M-cell field, 446 MB allocated and 596 ms for ONE plan; after 860 x 840 field, 9 MB and 16 ms. At the
20480 x 20480 the reader used to accept the same plan would have allocated about 5.4 GB.

## Roam (fully known maps)

When the world source reports `FullyKnown()` (d2dworld), `Explore` roams coverage points instead of revealing unknown cells; see internal/nav/roam.go and docs/d2d-pace.md section 'roam'. `Navigator.Coverage()` returns visited/total points. The Game.exe path is unchanged.

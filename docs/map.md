# d2map: world map reader (agent MAP)

## API summary (package `internal/d2map`, stable names)

All coordinates are absolute game subtiles; `Rect` is half-open `[X,X+W) x [Y,Y+H)`. One tile = 5x5 subtiles.

- `Grid`: `NewGrid()`, `ParseASCII(x0,y0,rows...)` (tests), `SetBlock(Rect, []uint16)`, `Known(x,y)`, `Flags(x,y) (uint16, ok)`,
  `Walkable(x,y)` (known and BlockWalk 0x1 clear; unknown = not walkable), `Bounds()` (storage bbox, empty if nothing known),
  `Region(x0,y0,x1,y1)` (half-open copy), `Clone()`, `KnownCount()`, `Version()`, `ASCII(Rect, marks)`.
  Flag consts `BlockWalk ... DeadBody`, `DynamicMask`.
- `Tracker`: `NewTracker(mem.Source)`, `Update() (State, error)` once per tick (error wraps `ErrNotInGame` in menus, loading
  screens, level changes; cached state survives), `Current() *LevelMap`, `Level(no)`, `LevelNos()`, `RescanEvery`.
  The player's room and its neighbours are re-read every tick, other known rooms every `RescanEvery` ticks (default 4).
  Rooms of every level the client has loaded are routed to that level's own grid. An Act map-seed change (new game) clears the cache.
- `LevelMap`: `No, Rect, Grid, Presets, Exits, Adjacent, Warps, Rooms`, `Waypoints()`, `BorderCells(levelNo)`.
- `State`: `Pos, LevelNo, Act, Room, NearRooms, LevelChanged, NewRooms, ActiveRooms`.
- Static: `Preset{Kind(1 npc,2 object,5 tile), ID, Pos, Room2}`, `Exit{Dest, Pos, Room2}`, `Adjacent{LevelNo, Rect, Contact}`.
- Low level: `Reader{S}` with `Player, Act, Levels, Level, LevelRoom2s, Room2, Room1, ActiveRoom1s, Coll, Presets, RoomTiles`.
- Fixtures: `Recorder` (wraps a Source, remembers every byte read), `Image` (replayable `mem.Source`, `Save`/`LoadImage`).
- Tool: `cmd/d2mapdump` (`probe|dump|explore|explore2|movetest|record|render|watch`); `render FILE` works offline.

NAV notes: unknown cells are not walkable; the Grid never "completes" a level, so plan only over known cells and re-plan as the
grid grows. Dynamic bits (`DynamicMask`) describe monsters/players/items, not terrain; Kolbot's rule (BlockWalk only) is `Walkable`.

## What was verified live (Rogue Encampment, 1.14.3.71, 2026-10-28)

VERIFIED LIVE = read from the running client and cross-checked (hexdumps and the unit tables).

- Chain player(+0x1C pAct, +0x2C pPath) -> Path(+0x02 x, +0x06 y words, +0x1C pRoom1) -> Room1 -> Room2 -> Level: all as in docs 3.2.
- Act: +0x0C seed, +0x10 pRoom1 (head of active list, chained by Room1+0x7C), +0x14 act index, +0x48 pMisc. ActMisc +0x47C pLevelFirst; Level +0x1AC next. VERIFIED.
- Level: +0x10 pRoom2First, +0x1C/+0x20 pos and +0x24/+0x28 size in TILES (town: 1192,968, 56x40 tiles = 280x200 subtiles), +0x1D0 level no (1). VERIFIED.
- Room2: +0x08 near array, +0x24 next, +0x2C near count, +0x30 pRoom1, +0x34..0x40 pos/size in TILES, +0x58 pLevel, +0x5C pPreset. VERIFIED. Rooms are 8x8 tiles (40x40 subtiles); the town has 35 Room2s.
- Room1: +0x00 near array, +0x10 pRoom2, +0x20 pColl, +0x24 near count, +0x4C..0x58 pos/size in SUBTILES, +0x7C next. VERIFIED. +0x74 pUnitFirst held a plausible pointer, not exercised.
- CollMap: +0x00..0x0C pos/size subtiles (equal to Room1 +0x4C..), +0x10..0x1C the same in tiles, +0x20 pMapStart (WORD*, row-major, W wide). VERIFIED. `pMapEnd` is +0x24 by layout (the +0x22 in D2BS is a typo); not needed, the reader bounds by W*H.
- Presets: PresetUnit +0x04 txt no, +0x08 posX, +0x0C next, +0x14 type, +0x18 posY; absolute = room2 pos*5 + preset pos. VERIFIED: all 17 objects of the server unit table sat exactly on their presets (205 fence objects too); static NPCs d=0, wanderers drift 6-15 subtiles.
- Object classes seen: 119 waypoint at (6044,4909) (matches Act 1 town; not in the unit tables because it is far from the player), 267 stash (6016,4934) (unit table agrees), 37 torches (walkable cell, Object bit), 39 fire (BlockWalk), 40/41/42/65 the east fence (x 6120..6140, BlockWalk). 78 and 385 unidentified.
- NPC classes: 147 Gheed (5985,4983), 148 Akara (6082,4911), 150 Kashya, 154 Charsi (5984,4922), 155 Warriv, 149/152/179 guards/wanderers. Layout matches the known Act 1 town (Akara NE, Gheed/Charsi W, waypoint N-centre, stash centre).
- Preset lists and tile lists exist only for Room2s that have room data: rooms without a Room1 have presets=0. So static info accumulates as rooms load, same as collision. Tracker rescans when pRoom1/pPreset/pRoomTiles change.
- Collision bits seen in town: 0x1 solid (32030 cells), 0x4 wall (with 0x1), 0x10 alternate tile (floor decoration, centre of town, NOT exits), 0x80 player (the player marks a small cross of cells, moves with it), 0x100/0x1000/0x2000 NPC bits (0x1100 Kashya/Warriv, 0x2100 guards), 0x400 object (torch/stash cells stay walkable), 0x401 object+solid. Matches the docs bit table.
- Grid updates live: walked the character 8 subtiles; the 0x80 cross moved with it in the merged grid on the next tick.
- Active rooms: 35-36 Room1s are active in town (all rooms with data, plus a column of Blood Moor rooms at x=5920); four town rooms at x=6160 had no room data and stay unknown (48000 of 56000 cells known). Tracker.Update costs about 150 small reads.
- Fixture: `internal/d2map/testdata/rogue_encampment.d2mi` (18 KB gzip, 150 KB of memory, includes unit tables).

## Exits: what is and is not verified

- The town has NO tile presets (type 5) and no RoomTiles. Outdoor level links are plain level borders: Room2.near contains Room2s of the neighbouring level (adjacent level 2 = Blood Moor, rect 5480,4800 480x280). The Blood Moor exit is the gap in the west wall at x=5960, y=4945..4959 (15 cells; plus one walled-in stray cell at (5960,4867)). `LevelMap.BorderCells(2)` finds it. VERIFIED LIVE.
- Type-5 tile presets + RoomTile (+0x4C pRoomTiles, +0x04 next, +0x00 dest Room2, +0x10 nNum*) matching for caves/stairs/dungeons: UNVERIFIED on this client (town has none; leaving town is forbidden). Implemented from MAPAPI and tested only on hand-built memory.
- `Level.WarpX/WarpY` (+0x1E0/+0x204): town holds one entry (6060,4900) that is near the waypoint, not the gate. Semantics UNKNOWN; exposed as `Warps`, do not use.
- Waypoint object id list is from objects.txt/Kolbot: only 119 verified.

## Dead ends / gotchas

- Collision cells with Object bit still walkable (cell of a torch/stash centre); the real footprint blocks adjacent cells. Use BlockWalk only.
- Walking east of x=6120 is fenced off; the town's reachable area is x 5960..6120, y 4895..5030.
- A click ordered while another agent is driving the character does nothing; the game is shared.

## Rogue Encampment (1 char = 2x4 subtiles; x from 5960, rows from y=4888; G Gheed, A Akara, K Kashya, C Charsi, V Warriv, W waypoint, S stash, @ player, n other NPC, o object, ~ walkable with flags, # solid)

```
################################################################################o#o##
################################################################################o#o##
...........................................................................##########
...........................................................................#####o#o##
......................n.....###...................#..n.........o...........#####o#o##
.....#..#######............o..............W.......#.......#..A.........n....####o#o##
...n....###..##....................~~~~~..........o..........................###o#o##
......#####.....o..................~~~~~..................o..................########
......#####.C.......~~~~~~~~.........#####...................................###o#o##
....................~~~~~~~~.........o.......................................###o#o##
........o...........~~~~~~~~.................................................###o#o##
....................n~~~~~~~S..........K.....................................###o#o##
...o........................................~~..........########.............########
..n.........................V...........................########.o...........###o#o##
........................o......~.o...o..................########.............###o#o##
..............o...............~~........................#######..............###o#o##
..........................................................#####..............###o#o##
...........n............................................o.....#..............########
..........................................~~~...............n..o.............###o#o##
.........................................o~~~.o.......~......................###o#o##
....o....................#####.....~~~~~~~~~~#.......~n......................###o#o##
........##.....o.........#####.....~~~~~~~~..................................###o#o##
........#####.............o....o.............................................########
........###.G...............~n...................................##..........###o#o##
........#....................................................................###o#o##
..........o.............................###.....@~...........................###o#o##
.....#...............n......###.........###.....~n..........n.......#........###o#o##
............................###.........###.o...~~................o..........########
..............................#.........###..................................###o#o##
.............................................................................###o#o##
########################################################################.....###o#o##
################################################################################o#o##
#####################################################################################
################################################################################o#o##
```

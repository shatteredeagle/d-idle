# Side-panel minimap v1 (TASK-148, sensesmap)

Design: `docs/research/info-unlocks.md` (tier 1). View only: the panel reads a snapshot, nothing it does reaches the bot.

## Data flow

d2map `Tracker`/`LevelMap` (bot goroutine) -> `senses.Publisher.Observe(lm, areaName, heroPos)` -> `Publisher.Snapshot()` -> `watch.Senses` -> `GET /api/senses` (poll about 2 Hz) -> `web/senses.js` canvas beside the video (top of `/watch`, not a tab since watchlayout).

- `internal/senses`: `Map` (schema `watch.senses/1`: areaId, area, version, x0, y0, step, w, h, rle, exits[], waypoints[]), `Snapshot` (map + hero + optional units, empty until TASK-151), `Build`, `Encode`/`Decode`, `Publisher`.
- Cells: 0 unknown, 1 walkable (BlockWalk clear, the Kolbot rule of d2map), 2 blocked. RLE is a flat `value,length` list in row-major order. A level longer than 256 subtiles is downsampled by an integer `step` (a block is walkable when any cell is).
- Fog of war is server-side: `Publisher` keeps the visited 4-subtile blocks of every area for the session (radius `senses.RevealRadius`, 24 subtiles) and masks cells, exits and waypoints the hero has not been near. The map `version` bumps on each new reveal or grid change; the page caches by version (`?v=`) and receives the map only when it changed. Unseen geometry is never sent.
- `Unlocker.Allowed(tier)` is the hook of the unlock ladder (TASK-149). Today `watch.FixedTier(n)`; n = 1 is the default, 0 is off. A locked tier's keys are absent from the JSON. A nil Unlocker is closed.
- Mount: `watch.ServeOptions.Senses` adds `/api/senses` and `/watch/senses.js` to the viewer, behind the same gate as the other viewer routes (token or passcode; 401 without; GET and HEAD only).
- An area without geometry is an error naming the area id (`senses: no geometry for area N`), the previous snapshot is kept.

## Tags

- V (tests, offline): RLE round trip, golden JSON of a synthesized grid (`internal/senses/testdata/tiny.golden.json`), downsampling, fog reveal, two areas keep their own fog, tier gating, version cache, GET only, 401 without token, the recorded Rogue Encampment image of `internal/d2map/testdata` (live-recorded 2026-10-28) builds a 140x100 map at step 2.
- U: reveal radius 24 (a guess to tune); that hero and presets share the grid's coordinate space on both backends (see Coordinates: no Origin is applied, V by code, live d2d check open); exits/waypoints of a live level (the recorded town image gave none inside the reveal radius); the canvas look (no browser run was done).
- Not built: enemy dots, bars, elites (TASK-151/152), the ladder (TASK-149), video overlay.

## Wiring (done, TASK-148 AC1)

- Knob `senses.tier` (default 1, 0 = no panel; `internal/env/tuning_senses.go`, listed in docs/knobs.md). `cmd/d2bot` (`startLadder`, only with `-watch`) builds a `senses.Publisher` and sets `ServeOptions.Senses = &watch.Senses{Src: pub, Lock: watch.FixedTier(tier)}` when the tier is 1 or more; `useSenses` (cmd/d2bot/sensesfeed.go) then sets `Env.OnObserve` on the live and d2d backends.
- The call site is `nav.Navigator.observe`, the one place that runs `World.Update()` on the bot's goroutine for both backends (Tracker and d2dworld): right after a successful Update it calls `Env.OnObserve(lm, State.Pos)`. The hook reads only; `Publisher.Observe` builds the immutable `Snapshot` (fresh slices, hero by value) under its own lock, so the HTTP side never touches the Tracker. Throttle: unchanged area and grid within 500 ms are skipped; an area change or a grown grid goes at once.
- V (offline tests): `go test ./cmd/d2bot -run SensesFeed` (two fake areas: snapshot follows the hero, fog reveals, version changes only on reveal or area change, fog kept per area, throttle, concurrent pollers) and `go test -race` of it; `go test ./internal/nav -run ObserveCallsHook` (hook receives the level map and tracker position, observation unchanged).

## Coordinates (the Origin question)

- Finding, V by code reading (not by a live run): both backends place grid cells, `State.Pos`, presets and exits in ONE space per level. Game.exe: absolute act subtiles, `LevelMap.Origin` is zero. d2d (`d2dworld.OriginNone`, the default): level-relative everywhere, and `LevelMap.Origin = 5*OriginX, 5*OriginY` is set only so static absolute tables (nav's known waypoint positions) can be converted; nothing in the grid or `State.Pos` is shifted by it (`internal/d2dworld/world.go`, `internal/nav/waypoint.go`). The panel therefore applies NO Origin: the hero is `State.Pos` (the position d2map itself uses against the grid), never the raw `Snapshot.Player` x/y. Test: `strip` maps with a nonzero Origin keep `X0` at the grid's own coordinates (cmd/d2bot/sensesfeed_test.go).
- U: the live d2d run (hero cross lands on the right cell on a real d2d level, `OriginTiles` mode if ever enabled: then Pos and grid are both absolute and the same argument holds). The recorded Rogue Encampment image is Game.exe (Origin zero) and cannot show the d2d case.

## Live checklist (ready to run, not run)

1. Start a hidden instance with `d2bot ... -ladder -watch -watch-video` (senses.tier defaults to 1).
2. Open `/watch?t=<token>` (video and map sit at the top): the panel shows "Waiting..." then the town map around the hero only.
3. Walk: fog clears behind the hero, the cross follows within 1 s, zoom +/- and drag work.
4. Take a waypoint/exit: the map changes (new area name), the previous area keeps its fog on return.
5. Without the token `/api/senses` answers 401. Writes after start: 0 (read-only).

## Contiguous areas (watchlayout)

User feedback 2026-11-08: the minimap showed only the current area. Now the last visited neighbours are drawn in the same canvas.

- Table: `internal/senses/adjacency.json` (embedded; sits there because `go:embed` cannot reach `data/`). It lists which area pairs may be stitched: 1-2 (V: both recorded town images report area 2 as the only neighbour, contact on the whole shared edge), 2-3, 3-4, 4-5, 5-6, 6-7 (S, the Act 1 wilderness chain, no image). A pair not listed is never stitched (dungeon entrances and portals are warps, not walks). The offset is NOT in the table.
- Alignment is derived: `Publisher` keeps the last fogged map and level rectangle of each visited area. For a listed pair it finds the neighbour's rectangle in the current level's own `Adjacent` list (`LevelMap.Adjacent`, from `pRoom2Near`) and sets `dx,dy` = that rectangle minus the one recorded when the neighbour was visited. Both frames absolute (Game.exe): dx=dy=0 (V on the recorded town: Blood Moor `{5480,4800,480,280}` ends exactly at the town's west edge x=5960). Level-relative frame (d2d without AddOrigin): the difference carries the shift (synthetic test only; U live).
- Fallback ("not stitched"): no table, pair not listed, the level reports no such neighbour, rectangle size mismatch, or a neighbour grid over `senses.MaxNeighbourRLE` runs: the neighbour is left out and `Publisher.Skipped()` holds the reason, which names the pair (`senses: adjacency 2<->1: ...`). At most `senses.MaxNeighbours` (2), most recent first.
- API: `/api/senses` gains `neighbours: [{areaId, version, dx, dy, map?}]`, same `Map` schema, fog as of the last visit (each area keeps its own fog). `map` is omitted when the page sends `nb=<areaId>:<version>,...` with that version. Only with tier 1 (the same gate as the map); tier 0 or a units-only unlock sends none.
- Panel: one world canvas in subtiles; neighbours are drawn dimmer behind the current area, the zoom-0 view fits the union box.
- V (offline): `go test ./internal/senses -run 'Stitched|Relative|Fallbacks|Adjacency|Cap'`, `go test -race ./internal/watch -run Stitch`, `go test ./internal/watch -run SensesJS` (node, when installed), golden `internal/watch/testdata/senses-stitched.json`. The Blood Moor grid in the tests is SYNTHETIC (no Blood Moor image was recorded); only the town image and its Adjacent rectangle are real.
- U (live check item): walk town -> Blood Moor -> town on d2d and look for (a) the neighbour abutting the current area without a gap or overlap, (b) the reported `Adjacent` rect of the town from inside the Blood Moor on d2d. Until then d2d alignment is unverified; a wrong one shows as a visible seam, never as a crash.

## Area names (TASK-187)

`internal/status/areas.go` names every Act 1 area id 1..39 with the in-game names (Burial Grounds 17, Crypt 18, Tower Cellar Level 1..5 = 21..25, Catacombs Level 1..4 = 34..37, Tristram 38). Status: id 17 V live (d2d panels run, 2026-11-08); the rest S (Levels.txt order, equal to internal/plandata/areas.json and the goals catalog; no extracted Levels.txt in this checkout). An id without a name reads `Unknown area (id N)`. Test: `TestAreaNameAct1Complete`.

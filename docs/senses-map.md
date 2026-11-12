# Side-panel minimap v1 (TASK-148, sensesmap)

Design: `docs/research/info-unlocks.md` (tier 1). View only: the panel reads a snapshot, nothing it does reaches the bot.

## Data flow

d2map `Tracker`/`LevelMap` (bot goroutine) -> `senses.Publisher.Observe(lm, areaName, heroPos)` -> `Publisher.Snapshot()` -> `watch.Senses` -> `GET /api/senses` (poll about 2 Hz) -> the `map` panel (`web/panels/map/panel.js`) in the stage band of the shell, beside the video.

- `internal/senses`: `Map` (schema `watch.senses/1`: areaId, area, version, x0, y0, step, w, h, rle, exits[], waypoints[]), `Snapshot` (map + hero + optional units, empty until TASK-151), `Build`, `Encode`/`Decode`, `Publisher`.
- Cells: 0 unknown, 1 walkable (BlockWalk clear, the Kolbot rule of d2map), 2 blocked. RLE is a flat `value,length` list in row-major order. A level longer than 256 subtiles is downsampled by an integer `step` (a block is walkable when any cell is).
- Fog of war is server-side: `Publisher` keeps the visited 4-subtile blocks of every area for the session (radius `senses.RevealRadius`, 24 subtiles) and masks cells, exits and waypoints the hero has not been near. The map `version` bumps on each new reveal or grid change; the page caches by version (`?v=`) and receives the map only when it changed. Unseen geometry is never sent.
- `Unlocker.Allowed(tier)` is the hook of the unlock ladder (TASK-149). Today `watch.FixedTier(n)`; n = 1 is the default, 0 is off. A locked tier's keys are absent from the JSON. A nil Unlocker is closed.
- Mount: `watch.ServeOptions.Senses` adds `/api/senses` and `/watch/senses_units.js` to the viewer (the map panel itself is served from `/watch/panels/map/` by the view handler), behind the same gate as the other viewer routes (token or passcode; 401 without; GET and HEAD only).
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

## Trail and coverage (TASK-296, maptrail)
- Trail (V offline): `senses.Trail` keeps hero samples (at most one per 250 ms, only when the position changed, 5 min, cap 1200). `/api/senses` adds `trail: [{ago ms, x, y, e?}]` (current area, oldest first; `e` marks a zone entry), `trail_area`, `ts`. The panel draws it under the hero (windows 30 s / 1 min / 5 min kept in localStorage `d2maptrailwin`), dots where a gap between samples exceeds 3 s (hover or tap shows the duration), ticks at zone entries.
- Coverage (V offline, U live): `nav.Navigator.CoverageView()` is a read-only copy of the roam points (`seen`, `failed`, current leg target, legs, held); `cmd/d2bot/sensesfeed.go useSensesCoverage` hands it to `Publisher.SetCoverage` from the OnObserve hook (nav goroutine). `/api/senses` adds `coverage: {area, cell, cover, cells:[[x,y,0|1|2]], total, target, state, legs, held}` only when the roam level is the hero's level. The `Coverage` button toggles the grid (off by default). Only roam state is shown; the frontier explore of explore.go has no accessor (U).
- Not verified: live bot, coverage and hero in the same coordinate frame (U, both level-relative by the code reading).

### Coverage cap fix (TASK maptrail2)
V live bug (phone, Blood Moor, 1593 points): `SetCoverage` kept the first 400 cells, and nav sorts points left to right, so only the west strip drew while the status line said 119/1593. Now the cap is 2000 and a larger list is downsampled uniformly (every n/2000-th point, `cell` grows by ceil(sqrt(n/2000)) so the picture still covers the zone). The payload carries `total`, `drawn`, `seen`, `failed` (counted over all points); the status line adds `drawing N of M` when downsampled. `cell` is `nav.roamSpacing` (7), the same spacing the points are sampled at (V by code; the screenshot shows contiguous squares). Test: `TestCoverageSpansWholeZone`.

## History: activity-coloured trail and event markers (maphistory)
- Activity (V offline, U live): each trail sample carries an `a` code (0 walk, 1 combat, 2 town, 3 farm, 4 flee, 5 rest; the names are `trail_legend`). `cmd/d2bot/senseshistory.go activityFor` maps the runner's whyline code (`runner.Status().Why.Code`) and `Fighter.Attacking()` onto it; unknown codes are walks. Loot walk-back has no signal of its own (U) and shows as walking; idle shows as the stall dots and the `stall` share.
- Events: ring of 500 events (30 min) in the publisher, `events: [{ago, x, y, k, t, big?, f?}]` for the current area, kinds kill, damage (life drop of 15 points or more between observations), potion, loot, levelup, death, fight. Kills during a fight get its summary `f` (`combat.FightSummary` through `Fighter.OnSummary`, the `end=` reason, kills, damage, min life, seconds). Chicken, portal and waypoint touches have no feed yet (U); chicken shows in the fight's `left`.
- Caps: trail 5000 points over 30 min (`TrailCap`; the older half is thinned to every second sample when full), events 500. `trail_window_s` is the longest window.
- Panel (manifest version 3): colours by activity with `--d-*` tokens, combat drawn thick with crossed swords where a fight starts, glyphs per event kind, `Events: key|all|off` and the window (30 s, 1 min, 5 min, 30 min) remembered per viewer, a share line `walk 41% . combat 33% . stall 5%` for the visible window, tooltip on hover or tap. No animation, so reduced motion needs nothing.
- Not verified: live bot and a phone; the uishots golden set needs `--update` for the map panel.

## Timeline and extra feeds (maphistoryA)
- `GET /api/history` (`watch.history/1`, same tier gate as the map; the panel gets it as the `history` key of `GET /api/senses?hist=1`, asked for every 10th poll while the timeline is open, because panels may not fetch): the sample ring, one sample per 2 s for 60 min (cap 1800): `{ago, l, m, a, w, lv, xp, ar}` (life and mana percent, activity code, whyline code, level, xp, area). Gold is not sampled: the runner status carries none (U).
- Panel (manifest version 4): `Timeline` shows an activity ribbon over the window, a range slider (left oldest, right now), Play/Pause and speed 1x/4x/16x. Scrubbing or replaying moves a ring along the trail and writes `3.0 s ago: combat (fight), life 62%, mana 40%, level 5` from the nearest sample (ages are rebased through the server clocks `ts`). Replay steps every 200 ms, every 1 s under prefers-reduced-motion.
- Feeds: chicken and town-portal markers come from `FightSummary.Left`; waypoint markers from the runner's waypoint hook (first touch in Normal only, as that hook); loot walk-back is activity 6 (`loot`, dashed green) from `Runner.LootWalking()` (V by code reading of walkBackLoot; U live).

## Heat layers (maphistoryB)
- `senses.Heat` keeps per-area counts in cells of `HeatCell` (7) subtiles, the roam coverage spacing: kills (kill events at the hero's cell), time spent (each observation adds its gap, at most 1 s, to the cell the hero stood on) and damage taken (life points lost, from the same drop that makes a damage marker). They persist for the whole game and reset for an area when its level map is a new object (a new game; U live, the nav coverage memory resets the same way). Cap 3000 cells per area, the busiest kept.
- Payload: `history.heat = {area, cell, kinds: [kills, time, damage], cells: [[x, y, kills, dwell s, damage]]}` for the current area, only with `GET /api/senses?hist=1&heat=1` (and `GET /api/history?heat=1`), so the 2 Hz poll stays small.
- Panel (manifest version 5): `Heat: off|kills|time|damage` (remembered per viewer), translucent squares of the same cell size under the coverage grid on a square-root scale, with a legend line. No animation.

## Fight cards and failure markers (maphistoryC)
- Fight card (V offline): click or tap a kill marker with a summary, or a combat segment of the trail, and a card lists when the fight ended, how long it lasted, the `end=` reason, kills, damage, lowest and final life, potions, casts, kites, approaches, dodges. Hostiles at start and at most are not in the fight telemetry (U), so the card has no such line.
- Failure markers, all from passive observers that change no behaviour: `nav.Navigator.OnFailure` (failed roam leg `roamfail`, mouth-gate hold `gatehold`, at the target point); `combat.Fighter.OnNote` (`ignore` at the monster with its window `s` seconds, drawn as a hatched circle whose arc is the share of the window left; `fallback` for a melee target fallback; `yield` when the fight found nothing and navigation yields). Repeats of one kind at one place within 5 s are merged. `key` events include roamfail, gatehold and ignore; fallback and yield show with `all`.
- Positions of ignored and fallback units are unit coordinates, assumed to match the hero frame (U live).

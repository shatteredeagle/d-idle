# Side-panel minimap v1 (TASK-148, sensesmap)

Design: `docs/research/info-unlocks.md` (tier 1). View only: the panel reads a snapshot, nothing it does reaches the bot.

## Data flow

d2map `Tracker`/`LevelMap` (bot goroutine) -> `senses.Publisher.Observe(lm, areaName, heroPos)` -> `Publisher.Snapshot()` -> `watch.Senses` -> `GET /api/senses` (poll about 2 Hz) -> `web/senses.js` canvas beside the video (Game tab).

- `internal/senses`: `Map` (schema `watch.senses/1`: areaId, area, version, x0, y0, step, w, h, rle, exits[], waypoints[]), `Snapshot` (map + hero + optional units, empty until TASK-151), `Build`, `Encode`/`Decode`, `Publisher`.
- Cells: 0 unknown, 1 walkable (BlockWalk clear, the Kolbot rule of d2map), 2 blocked. RLE is a flat `value,length` list in row-major order. A level longer than 256 subtiles is downsampled by an integer `step` (a block is walkable when any cell is).
- Fog of war is server-side: `Publisher` keeps the visited 4-subtile blocks of every area for the session (radius `senses.RevealRadius`, 24 subtiles) and masks cells, exits and waypoints the hero has not been near. The map `version` bumps on each new reveal or grid change; the page caches by version (`?v=`) and receives the map only when it changed. Unseen geometry is never sent.
- `Unlocker.Allowed(tier)` is the hook of the unlock ladder (TASK-149). Today `watch.FixedTier(n)`; n = 1 is the default, 0 is off. A locked tier's keys are absent from the JSON. A nil Unlocker is closed.
- Mount: `watch.ServeOptions.Senses` adds `/api/senses` and `/watch/senses.js` to the viewer, behind the same gate as the other viewer routes (token or passcode; 401 without; GET and HEAD only).
- An area without geometry is an error naming the area id (`senses: no geometry for area N`), the previous snapshot is kept.

## Tags

- V (tests, offline): RLE round trip, golden JSON of a synthesized grid (`internal/senses/testdata/tiny.golden.json`), downsampling, fog reveal, two areas keep their own fog, tier gating, version cache, GET only, 401 without token, the recorded Rogue Encampment image of `internal/d2map/testdata` (live-recorded 2026-10-28) builds a 140x100 map at step 2.
- U: reveal radius 24 (a guess to tune); that hero and presets share the grid's coordinate space on both backends (d2d level-relative `Origin` is NOT applied; check on d2d); exits/waypoints of a live level (the recorded town image gave none inside the reveal radius); the canvas look (no browser run was done).
- Not built: enemy dots, bars, elites (TASK-151/152), the ladder (TASK-149), video overlay.

## Not wired yet

`cmd/d2bot` does not construct a `Senses` or call `Observe`: the Tracker is single goroutine, so the call belongs in the bot's own tick after `World.Update()` (`internal/bot`, not owned by this card). Wiring is two lines: `pub := &senses.Publisher{}`; `ServeOptions.Senses = &watch.Senses{Src: pub, Lock: watch.FixedTier(1)}` (knob `senses.tier`, default 1, to be registered with the wiring); in the tick `pub.Observe(world.Current(), areaName, st.Pos)` about every 500 ms and on `st.LevelChanged`. Bot inputs are unaffected because Observe only reads.

## Live checklist (ready to run, not run)

1. Start a hidden instance with a bot built with the wiring above and `-watch -watch-video`.
2. Open `/watch?t=<token>` Game tab: the panel shows "Waiting..." then the town map around the hero only.
3. Walk: fog clears behind the hero, the cross follows within 1 s, zoom +/- and drag work.
4. Take a waypoint/exit: the map changes (new area name), the previous area keeps its fog on return.
5. Without the token `/api/senses` answers 401. Writes after start: 0 (read-only).

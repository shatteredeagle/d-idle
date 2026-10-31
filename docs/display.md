# Live display

`internal/status` (Go hub + panels), `internal/telemetry` (pure accumulators), `web/live/` (embedded UI, served under `/live/`),
`cmd/d2view` (demo with a simulated Sorceress). Try it: `go run ./cmd/d2view` then open the printed URL (`-speed N` scales sim time).

Mount in any binary: `hub.Mount(mux, "/live/")`. All page URLs are relative, so any prefix works.
Endpoints (relative to the mount): `/` page, `api/snapshot` (JSON of all panels), `api/events` (SSE), `manifest.json` (panel modules),
`api/panel/<id>/...` (a panel's own routes, e.g. map terrain).
SSE: first `snapshot` (full state + `seq`), then `update` (`{seq,id,data}`, whole-panel replace, only when changed) and `append`
(`{seq,id,item}`, for logs). Clients ignore messages with `seq <=` the snapshot's. A slow client is dropped and reconnects (EventSource) to a fresh snapshot.

## Add a panel (Go + JS, no core changes)
1. Go: `hub.Register(status.Panel{ID: "foo", Title: "Foo", Order: 70, Snapshot: func() any { return fooData() }})`.
   `Snapshot` is polled (default every 1s, set `Every`); unchanged JSON is not re-sent. Event-driven: call `hub.Push("foo", data)` or `hub.Append("foo", item)`.
   Bulky data: set `Routes` (an `http.Handler`) and fetch `api/panel/foo/...` from the JS.
2. JS: create `web/live/panels/foo.js` (the manifest lists `panels/*.js` automatically; add the file to `embed.go` only if you add a new top-level path):
   ```js
   import { h, clear } from '../lib.js';
   export default { id: 'foo', span: 6 /* of 12 columns on desktop */,
     render(el, data, ctx) { clear(el); ctx.aside.textContent = 'subtitle'; el.append(h('p', {text: data.msg})); },
     append(el, item, ctx) {} /* optional */ };
   ```
   `render` runs on every snapshot/update (keep DOM in `el._state` if you need to avoid rebuilds). A Go panel without a JS module shows raw JSON.
   `lib.js` offers `h()`, `bar()`, `tip()`, `lineChart()`, `qclass()`, formatters. Theme tokens are CSS variables in `style.css`.

## Data contracts (Go types are the schema; JSON tags in the structs)
- overview: `telemetry.Overview` + `areaName`. Fed by `telemetry.Tracker` (`Observe(Frame)`, `SetActivity`, `SetGoal`, `RunStart/RunEnd`, `Error`, `Info`).
- character: `status.CharacterData`, from `CharacterSource.Character()`.
- inventory: `status.InventoryData` (equipment map by slot key, inventory 10x4, stash 6x8, belt X=column Y=row, `beltRows`),
  from `InventorySource.Inventory()`; an optional `Namer` fills empty `Item.Name`.
- world: `WorldData` (position, units, exits, terrain revision) + terrain RLE at `api/panel/world/terrain`; from the `MapView` interface
  (`AreaID, Bounds, Walkable, Player, Units, Exits, Revision`). Coordinates are game subtiles.
- timeline: events (`telemetry.Event`, `seq` monotonic, streamed via `append`) and activity segments. Kinds: kill drop death levelup area error potion activity run info.
- metrics: per-minute series (5-minute average, per hour), per-area stats, last 50 drops.

## Telemetry rules (internal/telemetry, all unit-tested, clock injected)
Kill = a non-friendly monster seen alive whose mode becomes 0 or 12 (monsters first seen already dead are ignored). Drop = ground item id not seen in this area
(items present on arrival are baseline). Potion = belt count decrease outside town. Death = `Frame.Dead` false->true. Gold earned = increases of carried+stash.
Rates use sliding windows (5 min, 1 h) with a `partial` flag until the window fills, plus session. Memory is bounded (rings; monster and item id sets are pruned/capped).

## Adapter sketch (not wired; internal/d2 and internal/d2map are not merged, names below are guesses)
```go
// bot side, once per tick (~5-10 Hz):
p := d2.ReadPlayer(src)                       // area, level, exp, gold, mode, stats, skills
frame := telemetry.Frame{InGame: p.InGame, InTown: data.IsTown(p.Area), Area: p.Area, Level: p.Level, Exp: p.Exp,
    Gold: p.Gold, StashGld: p.StashGold, Dead: p.Mode == d2.ModeDead /* player dead mode: UNVERIFIED */}
for _, u := range d2.Units(src, d2.TypeMonster) { frame.Monsters = append(frame.Monsters,
    telemetry.Monster{ID: u.ID, TxtNo: u.TxtNo, Name: names.Monster(u.TxtNo), Mode: u.Mode, X: u.X, Y: u.Y, Friendly: u.IsNPCOrMerc()}) }
for _, u := range d2.Units(src, d2.TypeItem) { if u.OnGround() { frame.Ground = append(frame.Ground, telemetry.GroundItem{ID: u.ID, Name: namer.ItemName(u), Quality: u.Quality}) } }
frame.Belt = telemetry.Belt{Health: belt.Count(d2.Healing), Mana: belt.Count(d2.Mana), Rejuv: belt.Count(d2.Rejuv)}
tracker.Observe(frame)
// readers implement the panel interfaces directly or via a thin wrapper:
//   type charAdapter struct{ src mem.Source }; func (a charAdapter) Character() status.CharacterData { ... }
//   d2map.Level  ->  status.MapView  (Walkable from the merged CollMap grid; Revision = count of loaded rooms)
hub.RegisterTelemetry(tracker, areaName)
hub.Register(status.NewCharacterPanel(charAdapter{src}))
hub.Register(status.NewInventoryPanel(invAdapter{src}, dataNamer))
hub.Register(status.NewWorldPanel(mapAdapter{lvl}, areaName))
```
The bot's runner calls `tracker.SetActivity(state, detail)` on behavior changes (states with colors: town, exploring, fighting, looting, resting, dead, loading, idle),
`SetGoal`, `RunStart/RunEnd` around each run, and `Error` on failures.

## What each panel still needs from the bot
- overview: real `Frame`s; activity/goal/run calls from the behavior layer; real experience table (`telemetry.DefaultExpForLevel` is recalled, UNVERIFIED, levels 1-30 only; DATA agent: experience.txt).
- character: stats/resists/skills readers (skill ids, names, hotkeys, selected skills) from MEM; real class and difficulty.
- inventory: item reader (code, quality, sockets, ethereal, identified, grid position/size, properties text) and the DATA `Namer`; belt layout; the display does not read item properties itself.
- world: MAP reader behind `MapView`; unit names; exits with destination area; `Revision` bump when rooms load.
- timeline/metrics: nothing extra once the tracker is fed; area names from DATA; per-monster names for kill text.
Not verified: the UI was checked in headless Chromium (screenshots, desktop dark and 390px light) but not on a phone or in Waterfox; no live game data was shown.

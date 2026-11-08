# Information unlocks: senses for the watcher (minimap first)

> status: DESIGN NOTE, proposal only, 2026-11-07. Not a decision (no ADR, nothing in docs/decisions.md); nothing built. Origin: user idea 2026-11-07, clarified "our own overlay on the watch screen". Builds on ADR 0018 (Codex, tally, return screen) and docs/research/fun-ideas.md.
> Tags: V read in this tree (file named); S secondary (a doc's claim); U unverified or judgment. No offsets are invented here: every missing reader is listed as missing.

## 1. The idea in one paragraph

The watcher sees the hero through a video of the game. In the first hour that screen is poor: the same view, no context. We add progressive information "senses", drawn by OUR watch UI from the bot's memory reads, never by the game, that unlock as the hero (and the player) learn the world: first a minimap, then enemy dots, then health bars and names, then elite and boss markers, then loot and danger. Watchers get a richer screen over time, which is one more visible reward for the Codex, tally and level progress the loop already produces (ADR 0018).

## 2. Hard limits (what a sense must NOT do)

1. View only. Reads game state, writes nothing, feeds nothing back into the bot's decisions (the bot never reads `/api/senses`). A test asserts the bot's inputs are identical with all senses on or off.
2. Not a cheat look. The sense shows what the game's own automap or health bars show a human player; nothing beyond what the memory already holds for units the client has loaded (no unloaded rooms, no hidden objects). Tier 1 shows only area the bot has actually walked or the client loaded (the Tracker's known cells).
3. Token-gated like the viewer: same gate as `/stream` and `/api/thoughts` (internal/watch/gate.go, V); an unlocked tier is data in the response, a locked tier is absent (not hidden by CSS).
4. Generic world data, not per-class: the ladder, triggers and tier table are the same for every class and build (AGENTS.md build flexibility); the table is data, not Go.
5. Same on d2d and game.exe: the sense reads through the bot's own readers, not through a backend-only verb.

## 3. The tier ladder and triggers

| Tier | Sense | Trigger (proposal) | Why this trigger |
|---|---|---|---|
| 0 | none (video only) | start | today's screen |
| 1 | minimap of the current area, hero dot, waypoint and exits | first Codex entry of kind `area` (Blood Moor or Cold Plains first met) or clvl 2, whichever first | cheap, early, within the first 10 minutes; a map is the first thing a new watcher wants |
| 2 | enemy dots (hostile, alive, in range) | the first monster Codex first (Fallen, kind `monster`) | "once you have met one, you can see them" |
| 3 | health bars and names, per monster class | per class: unlocked when THAT monster's Codex entry exists (Fallen bar after the first Fallen); a class not in the Codex shows a plain dot | the user's wording; makes the Codex a map of what you can see; costs nothing extra in code |
| 4 | elite and boss markers (champion, unique, superunique, with mod names) | first Codex `champion` first, or the first superunique first (Corpsefire, Bishibosh) | the champion mod table is already a Codex data row (docs/codex.md) |
| 5 | loot markers (rare, unique, set, rune on the ground) | tally spend: a sink row "Eye for loot" (price from the ledger registry), or first `drop` first | a spend decision, not a free gift; fits the "see the next 2 or 3 purchases" rule |
| 6 | danger rating (a plain number or colour from monster level vs clvl and elite count) | tally spend | derived data (ADR 0011 readiness) shown, not new reads |
| 7 | hero overlays (life/mana orbs as numbers, resists, merc) | clvl 12 or an act boss first | late, low value for the first hour |

Rules: a trigger is a ledger or Codex event already in the system (never a new game read); tiers 1 to 4 are earned by play, tiers 5 and up by tally (the sink), so tally has one more way to be spent. Unlock is permanent per profile and persisted beside `codex-<profile>.json` (docs/codex.md, V file naming) as `senses-<profile>.json`. A hero's death does not relock. Tier 1 and 2 are cheap on purpose: they are the "second minute" reward in the pacing targets of fun-ideas.md section 7.

Ties: Codex first = trigger for tiers 1 to 4 (cards codexfirsts TASK-130, funwire TASK-140); tally spend = trigger for 5 and 6 (tallystrip TASK-129, tallynooverlay TASK-141); the return screen shows a line "New sense: minimap" in its firsts section (internal/watch/returnscreen.go, V exists) and the unlock itself raises no question (no thought; it is a notice).

## 4. A mock in text

Watch page, desktop: the video on the left as today. A right-hand side panel "Senses" below the thoughts. Tier 1: a square canvas, dark ground, the explored cells as lighter blocks, walls as lines, the hero as a white dot with a facing tick, waypoint a blue diamond, exits small arrows with the neighbouring area name ("to Cold Plains"). Tier 2 adds red dots. Tier 3: a bar under each dot and a name label on hover or when the dot is within 12 cells of the hero. Tier 4: elite dots larger with a gold ring (champion), purple ring (unique), crown (boss), the mod names in a tooltip. A locked tier shows as a greyed slot "Next: enemy dots, meet a monster first" (the tease is part of the reward). On a phone the panel stacks under the video.

## 5. Feasibility (V by reading the code)

| Need | What exists today | Missing |
|---|---|---|
| Area geometry | `internal/d2map`: `Grid` of walkable/known cells in absolute game subtiles, `Tracker` accumulating rooms, `Bounds()`, `Known`, `Walkable`, `Clone()` (V grid.go, tracker.go); `internal/nav.Field` is built from it (V field.go) | the Tracker is owned by the bot runner; there is NO export to the watch hub (no snapshot struct, no per-area cache for the minimap). Needs: a read-only snapshot (downsampled grid + version number so the page repaints only when it changes) |
| Hero position | `Unit`/path reads in `internal/d2` (V units.go, `HeroInfo` in watch/mapview.go carries area, not x/y) | x/y in the hub; add to the snapshot |
| Exits and waypoint | nav has exit and waypoint hints (internal/nav/exit.go, waypoint.go, V by listing); preset units in d2map (V doc.go, memory 3.2) | a plain list of exit/waypoint points per level for drawing |
| Monster positions, class, mode | `d2.Reader.Units(typ)` returns `Unit` with mode (V units.go); d2d `monsters` verb gives `CellX/CellY`, `HP` pair, `Boss`, `Mods`, `Name` (V d2dctl/parse.go) | on game.exe the same through `d2.Reader`; x/y fields exist on `Unit` but I did not trace which; mapping class id to display name is `data.DB` (V, codex.md) |
| Monster health | `MonsterInfo.LifePct/Life/MaxLife` (V units.go); the comment says damaged monsters are UNVERIFIED | live check that LifePct falls with damage on game.exe (U) |
| Champion/unique/boss, mods | `IsChampion`, `IsUnique`, `Enchants[9]`, `UniqueNo`, `Name` (V units.go); d2d `Boss`+`Mods` ids | mod id to display-name table (codex.md lists it as missing) |
| Loot on the ground | pickit sees ground items; d2d `ground` verb (V verbs.go) | a hub-facing list with kind and position; the game.exe path U |
| Transport | hub + `writeJSON` + SSE `/api/events` + MJPEG `/stream` (V http_view.go) | `/api/senses` does not exist |
| Gating | token gate + per-tier data (V gate.go) | the tier table and persistence |

Transport: one JSON endpoint `/api/senses` returning `{schema, tiers_unlocked, map:{version,...}, units:[...]}`; poll at 2 Hz from the page (simple, same as `/api/thoughts`), or SSE at 2 to 4 Hz later. The map part is sent only when its version changed (the page caches it by version); units are tiny. A locked tier's keys are simply absent.

Video overlay vs side panel: the video is a screen capture of the isometric game view (a tap screenshot on d2d, x11grab on game.exe; V tapcapture.go, capture.go). Overlaying markers on it needs the game's 2:1 isometric projection (screen = f(world - camera)), the camera centre (not the hero exactly during scrolling), and the capture's crop, scale and resolution, and it must stay in sync with a frame that is delayed by the stream. None of those is read or calibrated in the tree today (U: the projection formula and the camera source). A top-down minimap on a canvas in a side panel needs only a linear transform of subtile coordinates and no calibration, and fails soft (stale by a tick is fine). RECOMMENDATION: v1 is the side-panel minimap and dots; overlays on the video are a later, research-first card (`sensesvideo`), with health bars drawn on the video only once a calibration probe proves the projection.

d2d vs game.exe: the geometry reader (`d2map`) is memory-based and runs on both (V doc.go). The d2d `monsters` verb is d2d-only, so the common path must be `d2.Reader.Units` through the bot's `mem.Source` (U: that both backends expose the same `Unit` fields to the hub; verify in sensesenemies). The game.exe path is the gate backend per ADR 0015, so the senses reader goes through the bot's runner, not d2dctl.

Cost: S small (under a day for a card), M, L.

| Tier | Cost | Reason |
|---|---|---|
| 1 minimap | M | snapshot export, endpoint, canvas |
| 2 dots | S after 1 | units list into the snapshot |
| 3 bars and names | S | fields present; live check of LifePct |
| 4 elites | M | mod name table, rings |
| 5 loot | M | ground list on both backends |
| 6 danger | S | derived |
| unlock ladder | M | persistence, gating, Codex/tally hooks |
| video overlay | L, research first | projection, camera, sync |

## 6. Recommended v1 and order

v1 = tier 1 minimap in the side panel (card sensesmap), then dots and bars (sensesenemies), then elites (senseselites). The unlock ladder (sensesunlock) can be stubbed with all tiers on behind a profile flag so the senses are testable before codexfirsts is wired; the real triggers need codexfirsts and tally. Not on the two-week playable path (returnscreen, policysheet, tallystrip, codexfirsts, playsession2): priority low or medium, after the first playable chain.

## 7. Proposed decision line (for the coordinator; not written to docs/decisions.md)

"Information senses (2026-11-07 idea, proposal): the watch UI shows progressive view-only senses (minimap, enemy dots, health bars and names, elite markers, then loot and danger) drawn from the bot's reads in a side panel, unlocked per profile by Codex firsts (tiers 1 to 4) and tally spend (tiers 5 up); they never affect bot behaviour and are token-gated; video overlay is deferred behind a projection probe."

## 8. Open questions (for the user)

1. Is the minimap early enough at tier 1 (first area met), or should it be a clvl 5 gift?
2. Should locked tiers be teased in the panel (my proposal) or invisible?
3. Do tally-bought senses (5 to 6) fit, or should all senses be free and earned by play?

## 9. Cards (milestone m-4, Proposed, priority low or medium, after the first playable chain)

| Card | Task | Depends on |
|---|---|---|
| sensesmap (side-panel minimap v1) | TASK-148 | none (needs the d2map export it builds) |
| sensesunlock (ladder, persistence, API gating) | TASK-149 | TASK-148, TASK-140 (codexfirsts wiring), TASK-141 (tally) |
| sensesvideo (research: overlay on the video) | TASK-150 | TASK-148 |
| sensesenemies (dots, bars, names) | TASK-151 | TASK-148 |
| senseselites (champion, unique, boss) | TASK-152 | TASK-151 |

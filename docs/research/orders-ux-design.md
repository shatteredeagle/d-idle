# Orders UX: a tappable map instead of a form (design, before any code)

> status: DESIGN for user reaction (2026-11-05, user choice "design first, then build"). Nothing here is built. Clickable mock: [mocks/orders.html](mocks/orders.html) (single file, fake data; open it in a browser, use the debug bar for 3 scenarios; `#b` and `#c` in the URL start scenario B or C). No task card (TASK-20 notes).
> Parent docs: [watch-screen-design.md](watch-screen-design.md) (orders box, section 3 Bot integration, section 7 look), [automation-ladder.md](automation-ladder.md), [ADR 0016](../adr/0016-automation-ladder.md) (an order is the player telling the hero where to be; there is no direct control).
> Tags: V verified (code read or run), S secondary (veteran advice from the d2-sme agent, memory not fetched), U unverified.

## 1. What is clunky today (V: `internal/watch/web/watch.html`, `app.js`)
The Orders tab is one row: a kind select (Go to / Fight here / Come home), an area select (8 names), a minutes number box (1 to 240), "Give order", then a plain list. Problems: the player must know area names and pick them from a dropdown; the hero's position, level and the danger of an area are invisible while choosing; duration is a typed number (no "until level 9", no "until I cancel", `MaxOrderMinutes` 240); no progress, ETA or route after confirming; new orders always go to the FRONT of the queue (`GoalsOrders.Order` uses `Add(goal, 0)`), so a queue cannot be built in order; no warning when the hero would refuse.

## 2. The design in short
1. A schematic **map** of the act (nodes = areas, edges = exits, dungeons dashed) replaces the area select. Data-driven for five acts; Act 1 populated.
2. Tapping a node opens an **order sheet** (bottom sheet on a phone, centred dialog on desktop): what to do (Farm here, Clear then come home), how long (10 min, 30 min, 1 h, Until level N, Until I cancel), route breadcrumb, **Go now** or **Add to queue**.
3. **Hero now** panel: what the hero is doing for the order, progress bar, time left, kills and xp since the order, route breadcrumb, Cancel, Come home. **Up next** queue with move up, move down, cancel; "Rest in town" is one tap.
4. **Conflict with the hero's guard**: the sheet warns in plain words with the reason; the player may still confirm; the guard decides on arrival and the screen says so (section 5).
5. **Feedback**: toast on confirm, route breadcrumb steps tick off, a journal line when an order starts, ends or is refused.
6. Phone first (one tab strip: Orders / Thoughts / Game / Journal), desktop shows map, current order, queue, Thoughts and Game video at once.

## 3. The map
Nodes in the d.idle look (tokens of `web/style.css`, see mock). A node is a button (`role=button`, `aria-label` carries the name and every status in words). Layout is stored as data (x, y per node on a 3-column grid), not computed, so a hand-tuned layout survives.

Act 1 graph (S: d2-sme; U for the repo's exit table, which `nav` owns). Spine: Rogue Encampment, Blood Moor, Cold Plains, Stony Field, Dark Wood, Black Marsh, Tamoe Highland, Monastery Gate, Outer Cloister. Side branches (dashed = dungeon): Blood Moor to Den of Evil; Cold Plains to Cave, and to Burial Grounds to Crypt; Black Marsh to Hole and to Forgotten Tower to Forgotten Tower Cellars 1-5; Tamoe Highland to Pit. The monastery chain (Barracks, Jail 1-3, Inner Cloister, Cathedral, Catacombs 1-4) is the MAIN PATH to Andariel, not a branch; the mock folds it into one locked "Monastery line" node. Several dungeon levels (Cave 2, Hole 2, Pit 2, Crypt, Mausoleum, Cellars 2-5) are folded into their parent node in v1 (one node, "Cave 1 to 2"); the order names the first level and the existing goal catalogue names (`internal/goals/catalog.go`: "Crypt", "Cave Level 2", "Pit Level 1", "Forgotten Tower Cellar Level 5") are the finer targets for a later zoom. Waypoints (Rogue, Cold Plains, Stony Field, Dark Wood, Black Marsh, Outer Cloister, Jail 1, Inner Cloister, Catacombs 2; S) get a small mark and shorten the breadcrumb ("by waypoint").

Node data (proposed `GET /api/orders/map`, section 8): `id`, `name`, `act`, `x`, `y`, `kind` (town, field, dungeon), `waypoint`, `orderable`, `edges[]`, `veteran` (`range`, `why` strings from a data table, S), plus per-hero status.

**Status chips** (never colour only: each has a glyph and a word, and the border style differs):
| Status | Shown as | Fed by |
|---|---|---|
| hero is here | blue thick border, "&#9670; here" | the bot's current area (`GoalsOrders.Here`, already wired, V) |
| current order | amber thick border, "&#9654; order" | the active order's area (Hub order state, V) |
| queued | "#n" | the Hub order list, position in queue (V) |
| too hard now | warn border, "&#9888; hard" | `Runner.areaGuardWhy` / `towerGuardWhy` (V `internal/bot/routeguard.go`): the Forgotten Tower guard (clvl floor, merc, kit) and the Monastery road guard, plus `areaRulesWhy` (gear gate, "banned after repeated deaths", V `geargate.go`). Only these three sources exist today; every other area is "open". A veteran clvl range (S) is shown as ADVICE in the sheet, not as a status, so the map never claims a danger the bot does not enforce. |
| cleared | "&#10003; cleared" | Act 1 milestones the bot already logs (`cmd/d2bot/milestone.go`: den, blood raven, forgotten tower, andariel, V) for quest areas; for other areas U: needs a "fully explored" signal from nav. Per the veteran review "cleared" means explored (map revealed, boss and exit seen) for dungeons and boss dead for quest areas; pack-clear is the weakest meaning. The bot must not say cleared for an area it never fully walked. v1 shows the check only where a milestone says so. |
| not orderable yet | dimmed, "not orderable yet" on tap | `OrderableAreas()` (V `internal/watch/sim.go`: 8 areas today) |

A "Map / List view" toggle draws the same data as a list (for screen readers, small text and keyboard).

## 4. The order sheet
Opens on tap of an orderable node. Contents top to bottom: title "Send the hero to NAME"; one veteran line (range and why, S); a warning box if the guard says no (section 5); **What to do** chips (Farm here, Clear it then come home); **For how long** chips (10 min, 30 min, 1 h, Until level N with minus/plus stepper defaulting to clvl+1, Until I cancel); route breadcrumb; one summary line in words; buttons **Go now** (first in the queue; the current order waits behind it) and **Add to queue** (last; only shown when something is running). Tapping the town node gives "Rest in town" (confirm only). Esc and the scrim close it; focus lands on the first pressed chip.

Mapping to what exists (V `OrderToGoal`, `goals.Params`):
- Farm here for N min = `farm_area` with `UntilMins` (exists). Until I cancel = `farm_area` with all limits 0, which the goals code reads as endless (V comment on `Params.UntilKills`); `validateOrder` must accept `minutes=0` for it.
- Until level N: NOT in `farm_area` today (U). Cheapest honest option: the Hub ends the order itself when the status poll reports clvl >= N (it already polls status in `feedHub`), by cancelling the goal and writing "reached level N" to the journal. No goals change needed.
- Clear then come home: U ("cleared" has no signal). v1 maps it to farm with the chosen time plus a Come-home order queued right behind it (two goals); the label says "then come home" and the journal line says which stop ended it. A real "explored" stop waits for the nav signal.
- Rest in town = today's `home` order (idle goal, 1 min, U whether that makes the hero heal and restock; V the mapping, U the behaviour).
- "Fight here" disappears as a kind: tapping the hero's own node with Farm here is the same order (the node is marked "here").

## 5. Conflict with the hero's own guard (who wins)
Derived from the code (V by reading, no run): an order becomes a `farm_area` goal; the goal's walk (`travelTo`, internal/bot/goal.go) calls `routeAllowed` before every route, and `areaGuardWhy` closes the Forgotten Tower (areas 20-25, Normal only) while `task.TowerWhy` says so, and the Monastery road likewise; `routeAllowed` also refuses banned areas and, for a gated character, everything but the first field. Nothing in `OrderToGoal` or `GoalsOrders` bypasses it (no override flag exists). **So today the guard wins: the order is accepted, the goal runs, the walk refuses, the goal fails and the order shows `failed` with the guard's reason** (U exactly how the runner reports the refused walk: S from `loop.go` `errAreaGated`; to be checked by the slice-2 test against a fake bot).

Two caveats (V): the Forgotten Tower is not in `OrderableAreas()` today, so this conflict cannot even be ordered yet; and the ladder decision point `t0.area.toohard` (`towerWhy`, docs/t0-slice.md) lets a learned `go` lift the LEVEL part of the guard while the kit part stays. That is the ladder hook: **a confirmed order to a too-hard area is exactly the question the point asks**, so it should ask it, not bypass it.

Design: in the sheet the warning is plain words with the reason from the guard (mock: "No mercenary and clvl 6 is below 16. The hero skips the Forgotten Tower below that level. You may still confirm: the hero's own guard has the last word"). Confirm is allowed (the player may know better, and the answer teaches the ladder), after confirm Thoughts shows one card "You ordered the Forgotten Tower; the hero would normally skip it: go anyway?" with the ladder's `go / skip / lower` choices (the existing `t0.area.toohard` choices) and the usual learning counter. A `go` answer lifts the level block only (kit rules stay, as in the code). If the guard still blocks, the order ends `failed` with the reason and a journal line "Order refused by the hero's guard: no mercenary and clvl 6 is below 16". No new bypass is added; that decision stays with the ladder (open question 1).

## 6. Current order, queue, feedback
- **Hero now**: sentence ("Walking to Cold Plains (step 2 of 3)", "Fighting in Cold Plains"), progress bar with `role=progressbar`, "about N min left" (for time orders: elapsed over limit; for level orders: xp percent; "until you cancel": no bar, "runs until you cancel"), kills and xp since the order (diff of the status poll at the order start; U for the exact source of kills, the Hub already counts kills for the feed), route breadcrumb with finished steps in green with a word state (done, now) not colour only, **Cancel order**, **Come home** (one tap: becomes the next order at the front and the current one waits, as in the mock).
- **Up next**: numbered list; up/down buttons (44 px, labelled) instead of drag; cancel per row. At most 5 open orders (`MaxOpenOrders`, V). Reorder needs queue position support from the goal manager (`Add(g, pos)` exists, V; moving an existing goal is U: remove and re-add works if no state is lost).
- **After confirm**: toast ("Order given: Farm Cold Plains until level 9"), hero panel switches to "Walking", breadcrumb shows the path from the hero's area (route from `nav`, U whether the bot can name hops before walking; fallback: the static `path` in the map data, which is what the mock does).
- **When an order ends**: journal line (`Order done: Cold Plains until level 9 (112 kills, +4800 xp)`, `Order cancelled`, `Order refused by the hero's guard: reason`, `Order failed: reason`), a toast if the page is open, and the Hero-now panel falls back to "Following its own plan". Failed orders stay visible in Up next area as a struck line until dismissed.

## 7. Layout and accessibility
- **Phone (390 px, V in the mock by headless screenshot)**: tab strip Orders / Thoughts / Game / Journal. Orders tab, top to bottom: Hero now, map (full width, vertical scroll, node 104x44 units scaled to about 112x48 px), Up next. The sheet is a bottom sheet (max 92 vh, scrolls). No horizontal scroll.
- **Desktop (1280 px)**: 12-column grid: map left (5 columns, spans three rows), Hero now and Up next right, Thoughts and Game video side by side under them; the tab strip is hidden. The sheet is a centred 480 px dialog.
- Tap targets at least 44 px (chips, buttons, nodes, stepper). Contrast: tokens of `web/style.css` (both light and dark, the mock follows `prefers-color-scheme`). Status is never colour alone (glyph, word, border style, aria-label). Focus ring on nodes, chips and buttons; `Enter` and `Space` activate a node; `Esc` closes the sheet; the sheet is `role=dialog aria-modal`, the warning is `role=alert`; list view as the non-visual route; chips use `aria-pressed` and a check mark when on.

## 8. Implementation plan
Reuse (V): `POST /api/orders` (`kind`, `area`, `minutes`) and `POST /api/orders/cancel` on the separate answer route; the 5-open limit; `OrderToGoal`; goal event mapping. Today's `GET /api/orders` already returns orders, areas, kinds, maxMinutes.

New or changed:
- `GET /api/orders/map`: `{act, nodes[], edges[], hero{area, clvl, xpPct}}`, per-area `guard{closed, reason}` from a new callback `Config.AreaGuard func(area string) string` supplied by `cmd/d2bot` (the bot's `areaAllowed`; `internal/watch` must not import `internal/bot`). Static node and edge data lives in a new `internal/watch/mapdata.go` (or JSON), one table per act, Act 1 filled.
- `POST /api/orders`: add `pos` (`front` default / `back`), `until` (`{minutes}` / `{level}` / `forever`), `then` (`home`); `minutes` stays accepted. `validateOrder` and `Order` gain `Until`, `Then`, `Level`; `orderText` words them.
- `GET /api/orders` gains per-order progress for the active one: `kills`, `xpGained`, `elapsedSec`, `etaSec`, `leg`, `route[]`.
- `POST /api/orders/move` `{id, delta}` (slice 3).
- Hub: ends a level order on the status poll; writes the journal lines; raises the conflict thought (calls the existing ladder point, slice 2).
- Front end: `app.js` and `watch.html` replace the form with the map, sheet, hero panel and queue (no framework, `textContent` only as today). The mock's CSS and JS are the starting point (it is NOT shippable as is: fake data, debug bar).
- Orderable list: grow `act1Areas` with Hole, Forgotten Tower, Tamoe, Pit, Crypt, Cave 2 (names must match `goals.Catalog`; U the nav names).

Tests: `validateOrder` table (until forms, `then`), `OrderToGoal` (forever, then-home pair) against two builds (Sorceress, Paladin; the guard text is build data via the ladder, not hard-coded), `GoalsOrders` against the fake bot (guard refuses: order `failed` with reason; level order ends on a clvl poll), `/api/orders/map` golden JSON, headless screenshot at 390 and 1280 px (V method of this mock: google-chrome-stable --headless on the worker host).

**Slices** (estimates are for one agent, Go plus front end):
1. **Map and sheet, replaces the form (about 8 h)**: map data and `GET /api/orders/map` (hero, orders, static nodes, guard via callback), front end map + list view + sheet with minute chips and "Until I cancel", "Rest in town", Hero-now panel with time progress and Cancel, journal lines. Orders still front-insert. Enough to retire the clunky form.
2. **Level, conflict, queue order (about 7 h)**: "Until level N" (Hub-side), "Clear then come home" as a pair, `pos` back / Add to queue, conflict warning plus the ladder thought, orderable list grown (Forgotten Tower and others), kills/xp/ETA/route in `GET /api/orders`.
3. **Polish (about 4 h)**: reorder (`/move`), failed-order lines, "cleared" from nav explored signal if it exists, Acts 2 to 5 data tables when orderable, a11y pass with a screen reader.

## 9. Open questions for the user (recommendation first)
1. **May a confirmed order override the hero's guard?** Recommend NO: the order asks the ladder's `t0.area.toohard` question and the hero decides (ADR 0016 keeps the hero in charge; a learned `go` then lifts the level block). Alternative: an "I insist" override per order (more control, but it re-creates the Soaky deaths the guard was written for).
2. **Does "Go now" put the new order first (today) or should Add to queue be the default?** Recommend: two buttons as in the mock, "Go now" primary; the current order waits, it is not cancelled.
3. **"Clear it, then come home": is "farm N min then home" good enough for v1?** Recommend yes; a real "explored" end needs a nav signal and is slice 3 at the earliest.
4. **Fold dungeon levels into one node (Cave 1 to 2, Cellars 1 to 5) or show every level?** Recommend fold in v1 (about 18 nodes fit a phone); zoom later.
5. **Show the veteran clvl ranges as advice in the sheet?** Recommend yes, labelled "advice, not a guard", from a data table the d2-sme agent signs off, so the map never claims danger the bot does not enforce.

## 10. Not verified
Nothing here ran against the bot. The mock is fake data and proves layout and interaction only. U: that a refused guard walk surfaces as the order's `failed` state with a readable reason; the `idle` goal behaviour of "Come home"; whether the bot can report route hops before walking; the kills source for "since the order"; the nav names for the extra orderable areas.

Mock check (V): rendered with headless `google-chrome-stable` on the worker host at 390 px (scenarios A and C, sheet open) and 1280 px (scenario B); one text overflow in a node was found and fixed; no horizontal scroll at 390 px.

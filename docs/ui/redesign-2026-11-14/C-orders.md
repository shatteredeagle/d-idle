# C. Orders: one control surface

> status: design proposal (TASK-416, 2026-11-14), not built. Tags per line: V = the backend exists and was read in the code named, U = unverified or needs bot work. The glossary already has "Player command" (a request the player queues on the watch page; runs right behind safety). "Orders" is the name of the panel; a row in it is a player command. No glossary edit is proposed.

## 1. The decision: which surface is primary

The user's position (2026-11-14, as relayed): the map is the main command input only if the design concludes it should be. It was judged against what the player actually does: watches daily and gives a handful of commands (go to a zone, repair now or farm until repair, sell, stash and identify errands, revive the mercenary, pause, apply a gear set).

| Option | What it is | For | Against | Cost |
|---|---|---|---|---|
| 1. Map-first | Every command starts with a click on the act map; non-spatial commands hang off the town node or the hero marker | The picture of the world is the menu; fun; level bands and waypoints are visible while choosing | One of the eight daily commands is spatial. Repair, identify, pause and gear sets have no place on a map, so they would be hidden behind "click the town". Only Act 1 has nodes (`internal/watch/mapdata.go`: acts 2 to 5 are empty, V), so in Act 2 the primary input is blank. A map needs about 350 by 640 px to be legible; a phone cannot keep that on screen with the game picture. The queue still needs a list somewhere | Map data for four acts before it works at all; a second list anyway |
| 2. Command bar first | A text or button bar ("Go to...", "Repair", "Pause") with a palette | Fast for a keyboard user; tiny; works for every command, spatial or not | Choosing a zone from a text list loses what the map gives for free (where it is, how far, how dangerous, what is next). Weak for a friend's first run | Cheap, but throws away the best part of today's UI |
| 3. Hybrid (picked) | One Orders panel owns the queue, the state and the verbs. The act map is its place picker: clicking a zone is the fastest way to say "where", and the map draws the queue back as pins and a route | Every command has one home and one status list. The map does what a map is good at. Works in acts with no map data (the picker falls back to the list view that exists today). On a phone the panel is a row of four buttons and the map is one tap away | Two tiles show the same orders; they must read one source (section 6 gives them one endpoint) | One new read endpoint, one new panel, a restyle of the existing map card |
| 4. Contextual buttons on tiles only | What exists: Repair on the gear tile, intents on the Journey strip, errands on the inventory page | No new surface | It is the problem the user described: six places, five queues, no shared status | Nothing, and nothing improves |

**Picked: 3, the hybrid, with the contextual buttons of option 4 kept as shortcuts into the same queue** (the gear alert still has "Repair now" on it; pressing it adds a row to Orders). Reason in one line: the queue is the thing that must be unified, a queue is a list, and only one verb needs a map.

This is open question 1 (H-open-questions.md). If the user prefers map-first, the cost is the Act 2 to 5 map data and a larger default map tile; the Orders panel is still needed as the status list, so nothing in the build plan before slice S4 changes.

## 2. The two maps

| | Act map (`orders-map` today, `zone-map` in the new presets) | Local minimap (`map`) |
|---|---|---|
| Data | `/api/orders/map` (`watch.ordersmap/1`): nodes, edges, `status {here, order, hard, cleared}`, `range`, `why`, `waypoint` (V) | `/api/senses`: fog, trail, units, coverage (V) |
| Job | **Command input for place**, and the picture of the queue | **Watching**: where the hero is, what is around, what it explored |
| Sits | Next to the Orders panel | Next to the game picture |
| Click | A zone opens the zone menu (section 4) | Nothing new. It keeps zoom and the layers popover |
| Shows of the queue | Numbered pins on queued zones, a gold ring on the current target, the route from the hero through town to the target, a red mark with the reason on a zone whose order failed | The current zone's explored percent (`/api/hero.explored`, V) and, when the bot reports one, the walk target (U: no target field in `/api/senses` today) |

Why the minimap is not a command input: there is no order kind for a coordinate (V: `orders.go` has `go`, `fight`, `home`), "walk to this spot" would fight the bot's own pathing and safety rules, and the tile refreshes twice a second, which makes a click target move. The one local command that makes sense, "Fight here", is a button in the Orders verb row (the API kind `fight` exists, V; no button today).

The two relate by one mark: the node with `status.here` is the zone the minimap shows. In the Standard preset they sit side by side; in Simple the minimap is the only map on the first screen and the act map opens from the "Go to" button.

What the act map gains (all additive fields on `/api/orders/map`, schema stays 1 with new optional fields, or `watch.ordersmap/2` if a field changes meaning):

| Mark on a node | Source | Status |
|---|---|---|
| Level band as colour (below, in, above the hero's level) | `range` + `hero.clvl` | V today (drawn as text only) |
| Too hard, with the guard's reason | `status.hard`, `hardWhy` | V |
| Waypoint found | `/api/atlas` | V as a separate card; U merged into the node |
| Journey chapter here, and its band | `/api/campaign` `chapters[].area`, `band`, `state` | V as data; join by area name, U |
| Explored percent | `/api/hero.explored` | V for the current zone only; per-zone history U |
| Locked (act not reached, area not orderable) | `orderable` | V |

## 3. The model

A **player command** is `{intent, args}`. The Orders panel shows every open one, in the order the bot will run them, with one state vocabulary.

| Intent | Args | Spatial | Backend today | Needs |
|---|---|---|---|---|
| `go` | `area`, `until: {minutes N} | forever | {level N}` | yes | V `POST /api/orders {kind:"go"}`; minutes and forever V; until level exists in `unlockorderqueue.go` (`OrderQueue.Add ... untilLevel`) but `postOrder` refuses it ("until-level comes later") | wire until-level (small) |
| `fight_here` | `until` | here | V `{kind:"fight"}`; no button | UI only |
| `town` | - | town | V `{kind:"home"}` (an idle goal; U that it always walks home) | live check |
| `repair` | `mode: now | farm` (the server picks from live gold, as today) | no | V `POST /api/repair/request`, `/cancel`, `GET /api/repair` schema 2 `trip` | none |
| `farm_gold` | `on` | no | V `POST /api/farmgold {on}` | none |
| `identify` | `all` or item | no | V `POST /api/inventory/identify`, `buyid`, `cain` | none |
| `stash`, `take`, `gold` | item or amount | no | V `POST /api/inventory/{stash,take,gold}` | none |
| `equip`, `unequip`, `trash`, `mark` | item, slot | no | V invdesk queue | none |
| `gear_set` | set id | no | V `POST /api/inventory/sets/apply`, `/cancel` | none |
| `journey` | `chapter`, `do: go_now | skip | hold | resume` | chapter area | V `POST /api/campaign/intent` | none |
| `sell_now` | - | no | U. Marked items sell on the next town visit; nothing forces the visit. The errand trip (`errandTripStep`) is the mechanism | a small invdesk errand kind |
| `revive_merc` | - | no | U. The merc policy revives in town when gold allows; no player request | a request flag that forces the trip |
| `pause`, `resume` | - | no | Half: `bot.Runner.Pause()` and `safeToPause` exist (V `internal/bot/runner.go`, `loop.go`); no watch route | one desk and two routes; rule for pausing in the field |
| `skill_points` | - | no | none (`docs/skilltrees.md`: no write path) | out of scope |

Never in this model: any gear decision by the bot (`docs/gear-policy.md`). `equip`, `gear_set` and `trash` are the player's own clicks and only that.

### States (one vocabulary for every row)

`queued` (accepted, not started) -> `waiting` (wants to start; the reason says what it waits for: "a safe moment", "the town trip to finish", "20 min cooldown") -> `active` (with a phase: `going`, `in town`, `repairing`, `returning`, `farming 140 of 300`) -> `done` | `failed` (reason, Retry where the backend allows) | `cancelled` (by the player or by the bot, with the reason).

A second kind of row is not a command: **`blocked`** rows state why the plan cannot go on (`journey_blocked` with `goals.Status.JourneyBlocked`, V in the bot; the charter needed for a zone; a pact that forbids). They carry the buttons that unblock them and also appear in `needs-you`.

Mapping from today's words (no backend rename): orders `queued|active|done|cancelled|failed` (V); invdesk `queued|doing|done|failed|cancelled` (`doing` shows as `active`); repair `trip.state` `queued|waiting|going|repairing|returning|farming|done|cancelled|failed` (the middle five show as `waiting` or `active` with the word as the phase).

### Order of rows = the bot's real order

The panel does not invent a priority. It prints the ladder of `docs/inventory-actions.md`:

1. **Safety** (not a row you can cancel): when the bot is fleeing, healing or recovering a corpse, a one-line banner sits on top of the list: "Safety first: retreating to town. Your orders wait."
2. **Player commands**, in the bot's own order: item actions (identify first, then equip and trash), errands (stash, take, gold, Cain, scrolls), repair, then the zone order. Each row carries a "next" mark when it is the one that runs at the next safe moment.
3. **The plan** (one grey row, always last, not cancellable): what the hero does when the list is empty, from `/api/campaign` `decision.why` ("Farming Cold Plains for level 10 to unlock Blood Raven") or the Why line.

Because the order is the bot's, **drag to reorder is not offered** in v1. Zone orders are the only kind with a real queue (`MaxOpenOrders` 5, V); for them the choice is made when giving the order: **Now** (`replace: true`, cancels the open zone orders; today's only behaviour) or **Queue** (`replace: false`, V in `postOrder`; the queue is also an unlock-ladder feature in `unlockorderqueue.go`, see open question 5).

## 4. Giving a command

### From the act map (place)

Click or tap a zone. A small menu opens at the node (a bottom sheet on a phone). It replaces today's centred order sheet; the content is the same data.

```
Cold Plains                      clvl 6-12 . waypoint
"open ground, waypoint"          you are level 6: in range
-------------------------------------------------------
[ Farm here ]  30 min . 1 h . until level 8 . until I cancel
[ Go now ]     [ Add to queue ]
Journey: Blood Raven is next door (too early, floor 9)   [ Go now anyway ]
Route: Rogue Encampment > Blood Moor > Cold Plains
```

| Node | Menu |
|---|---|
| A field or dungeon, orderable | Farm here (duration chips; default is the last one used), Go now, Add to queue; the journey line when a chapter lives there |
| Too hard (`status.hard`) | the same, with the guard's reason inline in amber and the button reading "Ask anyway" (the guard keeps the last word, as today) |
| The zone the hero is in | Stay and fight (`fight_here`), Explore the rest (`go` here until explored, U), Back to town |
| Town | Rest in town, Repair, Identify all, Sell now, Revive mercenary: the town errands, because this is where they happen. They are the same buttons as the verb row, not a second implementation |
| Not orderable | name, "not orderable yet", and why if known. No toast |
| Charter needed | the price inline: "Above your range. Charter: 12 tally" and a Pay and go button. No `window.confirm` |

### From the verb row (everything else)

The Orders panel header carries at most five buttons; which five depends on state, so a button is never shown disabled without a reason next to it:

`Go to...` (opens the act map picker, or the list in an act without map data) . `Town` . `Repair` (reads "Farm until repair" when gold is short, exactly as `trip.button.label` says today) . `Pause` (reads `Resume` when paused) . `More` (Identify all, Sell now, Stash gold, Revive mercenary, Apply gear set >, Farm gold).

### From an item tooltip

The inventory tooltip's actions (equip, trash, identify, stash, take, mark) post to the same routes as today. What changes: the row they create is in the Orders list on `/watch` too, with its state, and the inventory page's own queue list becomes the same component reading the same endpoint. Nothing moves off the inventory page.

### From an alert

A `needs-you` row carries the one command that answers it ("Weapon broken, about 180 gold" -> `Repair now`). Pressing it adds the Orders row and the alert collapses to "Repair queued".

## 5. Rules

- **Priority.** Player commands run right behind safety. The panel says so in words when it matters ("waiting for a safe moment: 2 monsters near"), using the existing Why codes (`repair_wait`, `player_items`, `repair_farm`, `journey_blocked`).
- **Confirm only what cannot be taken back or costs currency:** trash an item; pay a charter; revoke all rules; "Go now" into a zone the guard calls too hard (inline, one extra click, not a modal). Nothing else asks.
- **Undo.** Every accepted command shows a toast for five seconds with Undo, which is Cancel on the queued row (cancel from `queued` exists for orders, invdesk and repair, V). After that the row's own Cancel works while the backend allows it: `queued` and `waiting` always; `active` for zone orders, repair (until the smith) and farm legs; never for an item action in `doing` (V `TestDoingCannotBeCancelledAndRestartFailsIt`).
- **Failure.** A failed row stays for 20 minutes with its reason in plain words and, where allowed, Retry (repair has a 20 min cooldown, V: the button then reads "Retry in 14 min"). It also raises one `needs-you` row.
- **Empty.** "No orders. The hero follows the plan: <decision.why>." Under it up to three suggestion chips built from state: the next journey chapter ("Go now: Blood Raven"), "Repair (1 broken)", "Identify 3 items". No chip when nothing applies.
- **Disconnected.** The verb row disables as a whole with one line ("No connection to the hero. Orders are kept and shown when it is back."); the reconnect banners of `docs/onboarding.md` stay as they are.
- **ETA.** Shown only where a real number exists: minutes left of a timed zone order (`goal.leftMin`, V), the journey level meter (`meter.etaSeconds`, V), the repair farm leg ("140 of 300 gold", V). A repair trip shows its phase, not a guessed time.

## 6. API (additive)

Nothing existing changes shape. `GET /api/orders`, `POST /api/orders`, `/api/repair/*`, `/api/inventory/*`, `/api/farmgold`, `/api/campaign/intent` stay and keep their goldens.

**New read endpoint `GET /api/commands`** (`schema: "watch.commands/1"`): the Hub joins the five sources it already holds (orders, the repair desk view, the invdesk queue, the gold-farm desk, the campaign view) into one list. No bot change (V that all five are bound in `internal/watch`; U the cost of the join, which must stay a cached read like `/api/hero`).

```json
{
  "schema": "watch.commands/1",
  "seq": 41,
  "safety": { "active": false, "text": "" },
  "paused": false,
  "rows": [
    { "id": "inv:44", "intent": "identify", "args": {"all": true}, "text": "Identify all (3 items)",
      "source": "inventory", "state": "waiting", "phase": "", "next": true,
      "why": {"code": "player_items", "text": "waiting for a safe moment"},
      "progress": null, "etaSeconds": null, "can": {"cancel": true, "retry": false}, "givenAt": 1731600000 },
    { "id": "repair:1", "intent": "repair", "args": {"mode": "farm"}, "text": "Farm until repair",
      "source": "gear", "state": "active", "phase": "farming",
      "why": {"code": "repair_farm", "text": "farming for repair: 140 of 300"},
      "progress": {"pct": 0.47, "label": "140 of 300 gold"}, "etaSeconds": null,
      "can": {"cancel": true, "retry": false}, "givenAt": 1731599000 },
    { "id": "order:12", "intent": "go", "args": {"area": "Cold Plains", "until": {"minutes": 30}},
      "text": "Farm Cold Plains for 30 min", "source": "map", "state": "queued",
      "can": {"cancel": true, "retry": false}, "givenAt": 1731598000 }
  ],
  "blocked": [
    { "id": "journey:blood_raven", "code": "journey_blocked", "text": "Journey blocked: Blood Raven needs level 9 (you are 7)",
      "actions": ["go_now", "hold"] }
  ],
  "plan": { "text": "Farming Cold Plains for level 10 to unlock Blood Raven", "etaSeconds": 1500 },
  "verbs": [
    { "intent": "repair", "label": "Farm until repair", "enabled": true, "reason": "need 160 more gold" },
    { "intent": "pause", "label": "Pause", "enabled": true, "reason": "" }
  ],
  "limits": { "zoneOrders": 5, "itemActions": 8 }
}
```

Times in the reply are epoch seconds; the page prints them in the viewer's local time (the project reports in EST; the page must not print UTC).

**New write endpoint `POST /api/commands`** on the answer listener, same gates as today (origin list, token, JSON, small body, unknown fields refused): `{"intent": "go", "args": {...}, "mode": "now" | "queue"}`. In v1 it is a dispatcher: it validates and calls the same Hub and desk methods the existing routes call, so there is one behaviour and two doors. `POST /api/commands/cancel {"id": "repair:1"}` does the same for cancel. Error codes are the existing ones (`charter-needed`, `active`, `cooldown`, `undamaged`, `pact`, `full`).

Goldens to add (fixtures are only added, never edited): `testdata/commands/view-{empty,three-one-failed,blocked,paused,safety}-{sorceress,paladin}.json`, a compat fixture `testdata/compat/commands-v1.json` once shipped, and the node DOM goldens of the `orders` panel for the same five states.

## 7. Keyboard and phone

| Key | Does |
|---|---|
| `g` | Go to... (opens the act map picker, focus on the hero's zone; arrows walk the graph, Enter opens the zone menu) |
| `t` | Town |
| `r` | Repair (whatever the button reads) |
| `p` | Pause or resume |
| `i` | Identify all |
| `o` | Focus the Orders list; up and down select a row; `Delete` cancels it; `Enter` opens its detail |
| `z` | Undo while the toast is up |
| `/` | Command palette: type "cold" or "rep" (the option 2 idea, kept as an accelerator, last slice) |
| `?` | The key list |
| `Esc` | Close the menu, sheet or palette |

Keys are off while a text field has focus and in overlay mode. Every key has a visible button; none is the only way.

Phone (390 px): the status line and the game picture stay on top; under the picture one row of four 44 px buttons (`Go to`, `Town`, `Repair`, `More`); under it the Orders list, one line per row (text, state chip, a cancel target at the end). `Go to` opens the act map as a full-height sheet; a zone tap swaps the sheet content to the zone menu. The bottom tab rail stays (Watch, Orders, Hero, Log).

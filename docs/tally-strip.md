# Tally strip: credits from events, the strip, the charter sink

> status: built offline 2026-11-07 and wired into the bot by TASK-140 (see Wiring), TASK-129 (fun loop, ADR 0018, [research/fun-ideas.md](research/fun-ideas.md) sections 3.4 and 13 card 3). NOT verified live: nothing here has met a real game or a real journal. Ledger and registry design: [currency.md](currency.md).
> Tags: **V** verified by a test in this repo (named); **U** unverified or a guess.

Code: `internal/currency/tallyrows.go` (rows, credits, quote, charter spend, summary), `internal/currency/tallyrows.json` (the data), `internal/watch/tallystrip.go` (event hook, strip view, order flow). Off in vanilla mode (the ledger refuses; the view says so).

## Sources (credits from observed events, no item)

The hook is `(*watch.TallyStrip).OnEvent(goals.Event)`; the data is `tallyrows.json`, the payer is a source row of `registry.json` (`tally_*`).

| event | credit | key (idempotency) | source |
|---|---|---|---|
| kill (class `normal`, weight 1) | 1 per 25 kills | `kills/<character>/<event id of the 25th>` | `tally_kills` (soft cap 120 per UTC day, above it sealed) |
| first kill of a monster kind | 5 | `first/monster/<name>` | `tally_firsts` |
| first visit to an area | 5 | `first/area/<name>` | `tally_firsts` |
| boss kill (class `boss`, weight 0) | 15, once per name | `boss/<name>` | `tally_boss` |
| quest done | 10 | `quest/<quest>/<difficulty>` | `tally_quest` |
| waypoint lit | 10 | `waypoint/<area>` | `tally_waypoint` (U: no goals event says it; the bot calls `TallyStrip.Waypoint`) |
| expedition completed (back in town after a field trip) | 2 | `trip/<character>/<event id of the arrival>` | `tally_trip` (cap 40 per day) |

Event id = `<run>:<goals.Event.Seq>`. An event without a sequence number is reported on the strip as an error and not credited. Replaying a journal into the same ledger books nothing (V `TestKillsFirstsBossAndReplay`, `TestTallyStripGolden`, both for two builds); firsts and quests are account-wide (a second hero does not earn them again). Unknown kill class: error naming the class (V `TestCreditErrors`).

## Sink: charters

A charter is the price of a "go" order to an area above the hero's recommended range; in range (or over-levelled) is free; other order kinds are free.

- Zone table: `tallyrows.json` `zones` (area, min, max clvl). U: the ranges are the veteran ranges of `research/mocks/orders.html` (the SME tagged them UNSURE). An orderable area missing from the table is refused with an error naming it, never priced free (V `TestQuote`).
- Price = `charter.base` (20) times the tier multiplier for how many levels the hero is below the zone minimum: up to 3 levels x1, up to 6 x2, more x4 (so 20, 40, 80). U.
- `(*TallyStrip).PlaceOrder(req, approve)`: without `approve` nothing is placed and the error wraps `ErrCharterNeeded`; the returned `Quote.Text` is the hero's warning for the order sheet or a question (decide is not touched). With `approve`: a short balance is refused before anything is placed; the order is placed through `Hub.Give`, then the spend row `charter/<run>/<order id>` is booked; if the spend fails the order is cancelled. Tally never goes negative, the same order is never charged twice (V `TestCharterSpend`, `TestCharterFlow`, two builds).

## Strip and summary

- `TallyStrip.View()` / `Handler()` (`watch.tally/1`, GET only): `balance`, `sealed`, `recent` (last 5 credits), `expedition` (delta of the trip under way, null in town), `last` (delta of the trip that just ended), `next` (the three cheapest buys: the charter and unspent registry upgrades). Golden: `internal/watch/testdata/tally-*.json` (V).
- For the return screen (TASK-127): `currency.Tally.SummaryFor(Window{From, To})` returns credited, spent, net and the lines with their reasons for any window; `TallyStrip.View().Last` is the same for the last trip.

## How to add a row

- A new credit: add the source to `registry.json` (`sources` and the `tally` currency's `sources` list), then the credit row to `tallyrows.json`; a new event kind also needs a case in `Tally.Credit` and the hook. `ParseTallyRows` rejects a row whose source is missing or pays another currency (V `TestTallyRowsErrorsNameTheRow`).
- A zone: add `{area, min, max}`. A price: edit `charter.base` or the tiers.

## Wiring (TASK-140, done)

- Built in `cmd/d2bot/funwire.go` `newFun` when the profile has a d.idle wallet (`--overlay-dir`, d2d backend): `currency.NewTally(wallet.Ledger, DefaultTallyRows(), <store profile>, <run id>)` and `watch.NewTallyStrip`. The run id is `<D2BOT_RUN or run>-<unix start>`: the manager's sequence numbers restart with the process, so a new process must not reuse an old process's keys. Towns are told by name (`status.AreaName` of the five town ids).
- `Fun.OnGoalEvent` calls `TallyStrip.OnEvent`. Waypoints: `Runner.SetWaypointHook` (internal/bot/waypoints.go) fires from `markTouched` the first time this runner touches a Normal waypoint; the save reader's older state is not reported (V by `TestWaypointHookFiresOncePerNormalWaypoint`, U live).
- `GET /api/tally` on the viewer (GET only); the page header shows `strip` and hides it when tally is off. Without a wallet (vanilla) the route answers the "off in vanilla mode" view.
- Orders: `Hub.SetOrderPlacer(strip)` puts `PlaceOrder` in front of `POST /api/orders` on the answer listener. In range and other kinds: free, 200. Above the zone range without `"approve":true`: 402 `charter-needed` with `charter {area, price, text}` (nothing placed, nothing charged); with it: charter booked after the order, 200; short balance: 402 `insufficient`. The page asks with `confirm()` and resends. While the hero's level is not yet known (before the first status poll) an order is refused as `bad-order` rather than priced on a guess.
- The return screen's tally section prints the strip, the last trip's net and its first five lines.
- Fixed on the way: the trip window started at a sub-second time while ledger stamps are whole seconds, so the first credit of a trip fell outside "this trip" (V `TestFeedHubFunLoop`).

## Not done

- Order slot 2 (`MaxOpenOrders` from a ledger value) and the first perk are not built; `next` lists registry upgrades that exist (unwired). TASK-129 AC 2 stays open for that part.
- Tally needs the d.idle overlay mode; a vanilla bot shows nothing credited by design.
- U: every amount and price above; the pace (60 to 75 tally in hour one) is not measured.

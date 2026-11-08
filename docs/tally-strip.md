# Tally strip: credits from events, the strip, the charter sink

> status: built offline 2026-11-07, TASK-129 (fun loop, ADR 0018, [research/fun-ideas.md](research/fun-ideas.md) sections 3.4 and 13 card 3). NOT verified live: nothing here has met a real game or a real journal. Ledger and registry design: [currency.md](currency.md).
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

## Not done

- Not mounted: nothing registers `GET /api/tally` in `serve.go`, the web page has no header strip yet, and `GoalsOrders.OnEvent` does not call `TallyStrip.OnEvent`; the order route does not call `PlaceOrder`; the bot does not build a `Tally`. Those files are outside this task's `owns`.
- Order slot 2 (`MaxOpenOrders` from a ledger value) and the first perk are not built; `next` lists registry upgrades that exist (unwired).
- U: every amount and price above; the pace (60 to 75 tally in hour one) is not measured.

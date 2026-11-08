# Order queue (TASK-161, unlockorderqueue)

Design note: `docs/research/unlockables.md` row 8 (a brainstorm), `docs/research/fun-ideas.md` (order slots as a tally sink), `docs/research/orders-ux-design.md` section 4 and 6 (queue, until level). Built and tested offline on fixtures. Tags: V verified by the named offline test, U unverified.

## What it is

Queue up to three orders that run in turn. Code `internal/watch/unlockorderqueue.go`, table `internal/watch/orderqueue.json`, ladder rows `orderqueue`, `orderqueue2`, `orderqueue3` (group ui).

## Slots (one number, owned by the ladder)

The number of orders that may WAIT behind the running one is the count of open slot rows (the table lists them in order; the first row gates the whole queue). It is not a separate counter.

| row | opens | or |
|---|---|---|
| `orderqueue` | Den of Evil cleared (`quest_stage` den_of_evil >= 2) | none |
| `orderqueue2` | Kashya's quest done (`sisters_burial` >= 3) | Charsi's quest done (`tools_of_the_trade` >= 3), or 30 tally |
| `orderqueue3` | Deckard Cain rescued (`search_for_cain` >= 3) | 75 tally |

The tally prices are the fun-ideas sketch (slot 2 at 30, slot 3 at 75), U: tuned from session records. Buying goes through the existing `POST /api/unlocks/buy {id}`: a ledger `spend` with the key `unlock/<profile>/<id>`, so a retry books once and a short balance is refused (V `TestOrderQueueBuySlotThroughTheLedgerOnce`). A bought slot 2 or 3 does nothing while the Den row is shut. The "Charsi event" is the Tools of the Trade quest (her Imbue reward); U whether the player would rather read another event.

## Sequencer

`OrderQueue.Tick` (from `Fun.Settle`) places ONE order at a time through the same placer as `POST /api/orders` (`Hub.place`): the charter flow, the 5-open limit and `Give` apply unchanged, and the bot's route guard still decides on arrival. It places the head only when the hero has no open order (a player's own order, "Go now" or a proposal's go runs first). When the order ends: done or failed (for example refused by the guard) the queue goes on; cancelled outside the queue stops the queue (hands off) until `resume`.

- Until level N: the order is placed as an endless farm and the sequencer cancels it when the hero's level reaches N (the design's cheapest honest option); the entry says "reached level N". Adding a level not above the hero's, with minutes, or on a rest order is a 400.
- Charter: an above-range `go` is refused at add time with 402 `charter-needed` and the price (nothing queued); with `approve` it is queued and paid when it is placed, by the same `TallyStrip.PlaceOrder` (a short balance then fails that entry with the reason).
- Away rule (safety): the queue stops placing when the hero dies (`EvDeath`) and when `OrderQueueConfig.Halt` returns a reason (the bot passes the Pact / low-life pause, `pact.Controller.Paused`). Stopped entries stay waiting; `resume` lets the queue go on and stops again at once if the reason still holds. The queue never cancels or overrides a guard.
- Idempotence: `key` on an add makes a retry return the same entry; a replayed quest event or death event changes nothing twice. The queue is in memory only (a restart forgets it; the slot rows are persisted by the ladder).

V `TestOrderQueueLockedUntilTheDenAndSlotsOpenInTurn` (two classes, goldens `testdata/orderqueue-*.json`), `TestOrderQueueRunsOrdersInTurn` (two classes), `TestOrderQueueStopsOnDeathAndResumes`, `TestOrderQueueHaltStopsAndGuardRefusalGoesOn`, `TestOrderQueueCancelsAndPlayersOrdersGoFirst`, `TestOrderQueueCharterFlowAndKey`, `TestOrderQueuePostGatesAndBadBodies` (token, method, origin, content type, unknown field), `TestOrderQueueTableAndMissingRows` (an unknown slot row is an error naming it).

## API

- `GET /api/orderqueue` (viewer): `watch.orderqueue/1`, `open`, `capacity`, `slots[]` (id, title, open, teaser and price while shut), `entries[]`, `stopped`, `balance`. Locked: the first slot's teaser only.
- `POST /api/orderqueue` (answer listener, same gates as the other answer routes): `{action: add|cancel|resume|clear}`; add takes `kind`, `area`, `minutes`, `until` (minutes, forever, level), `level`, `approve`, `key`. 403 `locked`, 409 `queue-full`, 402 `charter-needed`, 400 `bad-order`.

## Not done

- Wired in `cmd/d2bot` (TASK-185): `FunConfig.OrderQueue` with `Quote: strip.Quote` and `Halt` = the Pact controller's `Paused`; V `TestOrderQueueWiredThroughTheOrdersPath` (two classes: quest events open slots, orders placed in turn through `Hub.place`, a death stops it, resume goes on). No viewer panel (the data is the route). Live: nothing run against a game; the Den "cleared" signal is the quest stage 2 the bot's quest reader already emits (U live), and the slot events and prices are U.

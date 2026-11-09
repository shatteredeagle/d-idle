# Manual repair (TASK-226, repairmanual)

User decision 2026-11-09: repairing gear is not automatic at first. The player tracks it (inventory page, gear strip on the watch screen) and tells the hero to repair. Automatic repair (once an hour, every half hour, when an item breaks, always) comes later as unlockable features. Tags: V verified by the offline test named, U unverified (needs a live run), S secondary.

## What changed in the bot

- Before: `town.needRepair` repaired below the `town.repairBelowPct` knob (30), `NeedsTrip` sent the hero to town under `town.tripRepairPct` (20), the chore plans in `internal/bot` set `Repair` for every trip, and a Force plan skipped the need check.
- Now: the real bot installs a repair gate (`town.SetRepairGate`, `cmd/d2bot/repairwire.go`, called by `wireInv`, so both backends). With a gate the town repairs only when `repair.Controller.Permit` says so; the knobs no longer repair, a Force plan no longer overrides the gate, and a durability trip is made only for a mode (never for a one-off request). Without a gate (drills, other callers) nothing changed (V `go test ./internal/town`, the old repair tests are untouched; they need the game tables and skip here).
- A named run (`D2BOT_RUN`: gate, soak, drill) installs no gate: it keeps the legacy durability rule, like the policy sheet's class starts, so measured runs stay comparable (choice, see open choices).

## Controller (`internal/repair`, V `go test ./internal/repair`)

`modes.json` (embedded, validated by `ParseTable`): `manual` (default, no unlock), `break`, `hour`, `half`, `always` (each names an unlock row), plus the Pact-only hidden modes `low` (under 20 percent) and `never`. A mode is data: `onBreak`, `everySecs`, `always`, `belowPct`, `never`. An unknown or locked mode acts as manual, so a hand-edited policy file cannot buy a mode (V `TestLockedModeActsAsManual`).

- `Request()`: "Repair now". Refused with a reason (HTTP 409 + code) when no worn item is damaged (`undamaged`) or a Pact forbids repairs (`pact`). Stays `requested` until the town confirms durability rose (`done`); a failed attempt or too little gold keeps it requested with a note, and the next visit tries again. `Cancel()` withdraws it. State persists in `repair-ui.json` in the state dir (V `TestPersistence`).
- `Permit(facts)`: undamaged gear is never repaired. A pending request is honoured in every mode except a Pact's `never`; otherwise the mode decides. An `hour`/`half` mode repairs when the interval has passed since the last repair (the first time at once).
- Class-free: no class or build appears in the code; both Sorceress and Paladin run the same tests (V `TestRequestThenRepairedTwoClasses`, `TestManualNeverRepairsOnItsOwn`, `internal/town/repairgate_test.go`, the latter U here: it needs the game tables).
- Pinned and equipped gear is never swapped by repair logic: the gate only decides whether the existing Repair All click runs; nothing here equips, unequips or touches pins (`invdesk`).

## Policy row `repair` and the unlock ladder

Row `repair` of `internal/decide/policy_chores.json`: Manual (default), When an item breaks, Every hour, Every half hour, Always, and the two hidden Pact options. The default changes nothing on the sheet but its meaning is now manual. The four automatic modes are rows of `data/unlocks/ladder.json` (group ui, bought with tally through `POST /api/unlocks/buy`):

| mode | unlock id | price (tally) |
|---|---|---|
| When an item breaks | `repair_break` | 30 |
| Every hour | `repair_hour` | 50 |
| Every half hour | `repair_half` | 70 |
| Always | `repair_always` | 90 |

Prices are U (a guess: neighbours are alerts 60, loot markers 40, danger rating 80; more eager means dearer). Only Manual is open at the start. The sheet lists a locked option disabled with its teaser, price and an Unlock button; `POST /api/policy` for a locked option answers 403 `locked` (V `TestPolicyRepairModesLockedUntilBought`, both classes). `PolicySheet.SetUnlocker` connects the ladder; `decide.PolicyOption.Unlock` and `Hidden` are the new fields.

## Seeing it

- Inventory page: equipped items show `durability cur/max` (`status.Item.Dur/MaxDur`, filled by `cmd/d2bot/display.go` from `town.Durability`); a Repair block shows the mode, the cheapest cost (`town.RepairEstimate`, a lower bound), the request state and the Repair now / Cancel button.
- Watch screen: `web/gear.js` (one `<script>` line in `watch.html`; it inserts its own strip under the header, own scoped CSS) shows each worn slot as `slot cur/max`, amber under the warn percent (30, `modes.json` `warnPct`), red `BROKEN` at zero, the cost line, the request state and the button. Hidden until `GET /api/repair` answers.
- Alerts: rules `gear_low` (warn, 30 min cooldown) and `gear_broken` (alarm, 15 min) in `data/alerts/rules.json`, raised once when a slot enters the state (`Alerts.SetGear`, called from `Tick`), again only after it was repaired (V `TestGearAlertsOncePerChange`). They are free rules like death and low life (open choice: the old test said only those two may be free).
- Routes: `GET /api/repair`, `GET /watch/gear.js` (viewer); `POST /api/repair/request`, `POST /api/repair/cancel` (answer listener, `policyGate`: origin list, token or passcode, JSON only, 4 KB, unknown fields refused; V `TestRepairRoutesGateAndReplyWithReasons`).

## Pacts

`data/pacts/pacts.json` "Griswold's Debt" sets repair options `low` and `never`. They stay as hidden options and hidden modes so the Pact keeps working; under `never` the Repair now button is refused with `pact` and even a pending request is not honoured.

## Tests

V (run here): `go test -count=1 ./internal/repair ./internal/decide ./internal/watch ./internal/status ./internal/invdesk ./internal/unlock ./internal/pact ./cmd/d2bot`. The policy machinery tests that used the old repair rule options read `internal/decide/testdata/policy_repair_legacy.json` (the row as it was). U: `internal/town/repairgate_test.go` compiles and vets but skips without the game tables (`/tmp/d2mpq` on the worker host was absent).

## Open choices (safe defaults taken)

1. Default manual also for new and existing characters; named runs keep the legacy rule.
2. Prices of the four unlock rows (30/50/70/90) are guesses.
3. The two gear alerts are free (not behind the alerts row), because manual repair needs the player to be told.
4. "Repair now" waits for the next town visit; no trip is made for it (user decision). A later mode "go now" is a separate feature.
5. A mode that is bought cannot be un-bought; switching back to Manual is free.
6. `always` mode repairs at every visit when anything is damaged, `hour`/`half` count from the last confirmed repair (memory only across a restart: the stamp is in `repair-ui.json`).
7. The cost line is a lower bound (magic item stat costs are not modelled, see `town.repairEstimate`).

## Live checklist (nothing here ran live)

1. Start the bot with `--ladder --watch`. `/api/repair` lists the worn slots with `cur/max`; the strip shows under the header; the inventory page shows durability per item.
2. Let gear wear under 30 percent: strip turns amber and an alert line appears once; at zero it turns red BROKEN and the alarm alert shows.
3. With the default Manual, walk to town: no repair happens. Press Repair now: state `requested`; at the next town visit Charsi is opened, Repair All clicked, the state turns `done`, durability is full, no memory writes (`d2cheat ledger` empty).
4. Request with too little gold: stays requested with the note. Request with undamaged gear: refused.
5. Policy sheet: Every hour is disabled with its price; buy `repair_hour` with enough tally, choose it, a repair happens at a visit after an hour.
6. Equip an item from the inventory page (pins the slot), then repair: the item stays worn and pinned.

## Fix 2026-11-09 (live report: Repair now did nothing)

Cause not proven (U): the bot logged no request at all, so a click that failed at the browser (auth, origin, a non-JSON error) vanished silently, and a request that did arrive could lose its gold to the potions and portal scroll bought first. Changes: the request and cancel are logged (`repair requested by the player`); gear.js keeps the last answer on screen ("Not sent: <reason>" including the HTTP status for a non-JSON error, V `TestGearJSShowsFeedbackForRefusalAndHTTPError` in a stub DOM under node); a requested repair moves up behind the sale in the town chore order, before ammo, belt and shop (U here: `TestRepairRequestedIsPaidBeforeSupplies` needs the game tables); when gold is short the request stays open and shows "waiting for gold: need N more" (V `TestShortKeepsRequestOpenWithAmount`), with a town log line. Choice: gold is not held back from potions when the repair cannot be paid anyway.

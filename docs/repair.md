# Manual repair (TASK-226, repairmanual) and the repair trip (TASK-303, repairtrip)

> status: since 2026-11-11 "Repair now" is a TRIP, not a wait for the next town visit (user decision, section "The repair trip" below). Open choice 4 and the "Fix 2026-11-09" wait wording are history; the rest of this page (modes, unlock ladder, Pacts, controller) stands.

User decision 2026-11-09: repairing gear is not automatic at first. The player tracks it (inventory page, gear strip on the watch screen) and tells the hero to repair. Repair stays manual and is not an equipment decision (docs/gear-policy.md). Automatic repair (once an hour, every half hour, when an item breaks, always) comes later as unlockable features. Tags: V verified by the offline test named, U unverified (needs a live run), S secondary.

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
4. (Replaced 2026-11-11, see "The repair trip": "Repair now" makes a trip at the next safe moment.)
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

## The repair trip (TASK-303, user decision 2026-11-11)

Tags: V verified by the offline test named, S secondary, U unverified (nothing here ran live). Numbers are the `trip` block of `internal/repair/modes.json` (data, not knobs); the ask for a `town.repairTripMinDurabilityPct` knob has no code counterpart: the existing `town.tripRepairPct` (20) only drives the automatic modes and is unchanged, the button works whatever the percent (the player asked).

### The button (from LIVE pack gold, not stash, not stale)

`GET /api/repair` is now `schema: 2`: every v1 field is unchanged (V `TestRepairReplyKeepsEverySchema1Field` against `internal/watch/testdata/compat/repair-v1.json`), plus the `trip` object (`state`, `button {action,label,enabled,reason}`, `line`, `gold`, `estimate`, `need`, `kind`, `trips`, `maxTrips`, `progress`). Gold is `Player.Gold` of the newest snapshot. `need` = ceil(`town.RepairEstimate` x `trip.goldMult` (1.3)); the estimate is a lower bound.

| situation | button | line |
|---|---|---|
| gold >= need (or no estimate) | Repair now | - |
| gold < need | Farm until repair | `need N more gold (have X, estimate Y)` (N = need - X) |
| nothing worn damaged | off | reason `nothing worn is damaged` |
| Pact `never` | off | reason `a Pact forbids repairs` |
| a request failed or the bot cancelled one in the last 20 min | off | reason `... try again in N min` |
| a request is open | Cancel repair (Cancel while farming) | the phase line |

V `TestButtonStateFromLiveGold`, `TestButtonDisabledWithReason` (Sorceress and Paladin).

### Repair now (enough gold): the trip

`POST /api/repair/request` (same gate: origin list, token or passcode, JSON, 4 KB) opens ONE request (a second answers 409 `active`; 409 `cooldown`, `undamaged`, `pact` as before). The bot (`internal/bot/repairtrip.go`, `repairTripStep` runs once per goal step) then:

1. SAFE MOMENT (`repairSafeMoment`, pure, V `TestSafeMomentRule`): never mid-fight. A pack within the clear radius is finished first. Exception: a boss, unique or champion in the pack is finished too when life >= 50 percent, `trip.elitePotionMin` (2) healing potions and a way home exist (bounded at 3 waits). After `trip.safeWaitSecs` (60) with a hostile within `trip.portalClearTiles` (10) the hero steps away with the fighter's crowd retreat (`SafetyCrowdRetreat`, at most 6 steps); if no safe moment shows up in 4.5 waits the attempt is dropped for 90 s. A quest step that must not be interrupted (`noTP`) or a boss run waits without a bound. The phase `waiting` and the Why line (`repair_wait`) show the reason and the seconds.
2. LOOT: one pickup pass at the clear radius, then go (V `TestRepairTripQuietFieldBothClasses`).
3. LEAVE (`advanceTrip`, repair trip flag): re-checks no hostile within 10 tiles of the portal spot (the fighter's ignore list NOT applied), then `Town.ToTown`: an open portal nearby, else a scroll or tome, else (no scroll) the town's own walk home: neighbour field, waypoint, level chain (unchanged code, `internal/town/trip.go`). The cast takes about 2 s (S). No-portal areas fall back to walking (that is `ToTown`'s own no-scroll path).
4. IN TOWN: the existing chores plan (heal, sell per marks, identify, repair, 2 scrolls and potions as the kit plan wants). The repair vendor is resolved from data (`ActOfLevel(...).Repair`), no NPC name in the bot. Gold: a requested repair is moved up behind the sale; when the pack is short and the stash is not, `town.repair` withdraws the shortfall (to estimate x 1.3) through the gold mover first (U here: `internal/town/repairtrip_test.go` needs the game tables). Durability is logged before (trip start) and after (`repair trip done before=.. after=..`).
5. BACK: `BackToField` through the hero's own portal (S: existing code; waypoint or walk when it fails). If already in town the trip skips the portal and goes straight to the smith and ends there (V `TestRepairTripInTownGoesStraightToTheSmith`).
6. The request is cleared as `done`; a restart in `returning` also counts as done.

Failures: the smith's menu failing gets one more round in the same visit; then the request stays open and `queued` with the reason, and the next trip waits the 20 min cooldown (no portal spam: V `TestRepairTripSmithFailureKeepsTheRequestAndGuardsTheLoop`). No way home: the trip is given up after the town's retries, one trip counted (V `TestRepairTripNoWayHome`). A trip an emergency trip replaced, or one dropped at the cast because a hostile came close, does not count (V `TestRepairTripInterruptedStartsOver`, `TestRepairTripRefusesToCastNearAHostile`).

NOT done, said plainly: "re-check for a crowd before stepping through" the hero's portal back cannot look at the field from town (the monsters of the field are not in memory in town). What exists: the field gate before the return (`gateField`, kit) and, after arrival, the fighter's ordinary safety (crowd retreat, chicken, town portal). U whether that is enough; a live look at the arrival is on the checklist.

### Farm until repair (not enough gold)

The request is `kind: farm` with a target (stored in `repair-ui.json`). The adapter switches the existing Farm gold order on for it (`goldfarm.StartForRepair`, field `for: repair` in `goldfarm-ui.json`): the same safe zone one rung down, the same area-thrash and poverty rules. Target = ceil(repairNeed x 1.3) + `resource.KitCost` + `PovertyMargin` (`goldFarmFigures`, V `TestFarmUntilRepairThenTripThenOrderEnds`); progress counts pack plus stash gold (the town withdraws from the stash), line `farming for repair: 140 of 300`, Why `repair_farm`. When the target is met the request becomes a repair trip and the farm order ends itself; it does NOT go on forever like the plain button, and a plain Farm gold order the player gave is never ended by a repair (V `TestPlayerFarmOrderSurvivesRepair`). Pressing Stop farming on a repair-owned order cancels the request (one Stop means stop).

### Guards

Cooldown `trip.cooldownSecs` (20 min) between two trips of a request and before a new request after a failed one or one the bot cancelled (a request the player took back or a finished repair does not lock the button); `trip.maxTrips` (2) trips per request, then `failed`; one open request at a time; the estimate still unpayable at the smith after the farm leg (`trip.farmLegs` 1) cancels the request automatically with the reason ("still short of gold after farming (need N, have X)", V `TestUnpayableAfterFarmLegCancels`); a request that started with enough gold and meets a higher price gets one farm leg first. Cancel button while queued, waiting or farming (V `TestPlayerCancelHasNoCooldown`). State persists in `repair-ui.json`: a trip that was going or repairing starts over after a restart (a hero already in town goes straight to the smith), farming stays farming (V `TestTripPersistsAndRestarts`, old files load: `TestOldStateFileLoads`). A named run (`D2BOT_RUN`) installs neither the gate nor the trip.

### UI

Gear panel version 3 and the Inventory repair card read `trip`; a reply without it (an older bot) keeps the version 2 behaviour. DOM goldens: `internal/watch/testdata/gear_repairtrip_dom.txt` and `inventory_repair_dom.txt` (node), read the controller's own replies `internal/repair/testdata/view-<state>-<class>.json` (`TestViewGoldens`, UPDATE=1 rewrites), states: ready, farm, disabled (undamaged, Pact), queued, waiting, going, repairing, returning, farming, done, cancelled, cancelled by the bot, failed, plus an old bot. The old strip `gear.js` follows the same fields. No `.focus()` without `preventScroll` (V `TestPollingJSFocusKeepsScroll`), `go run ./tools/uilint` clean.

### Proposed text for docs/decisions.md (the coordinator writes it)

"2026-11-11 (user, with the SME's guards): Repair now is a trip. Button from live pack gold against the repair estimate x 1.3 (Repair now / Farm until repair / off with a reason). A repair trip leaves only at a safe moment (never mid-fight, bounded wait of about 60 s), casts a portal only with no monster within about 10 tiles, repairs at the act's smith and returns through its own portal; Farm until repair first runs the Farm gold order to repairNeed x 1.3 + kit + margin and ends with the repair. Guards: 20 min cooldown, 2 trips per request, auto cancel if still unpayable after a farm cycle, one request at a time. Replaces decision 4 of docs/repair.md."

### Live checks (none ran; also rows in docs/live-checks.md)

1. Gear low, gold under 1.3 x estimate: the button reads Farm until repair with the "need N more" line; press it: the log shows `repair requested by the player kind=farm` and `goldfarm: farming gold on user request`, the line counts up; at the target the hero makes the trip and the Farm order ends (the Why line changes to the town trip).
2. Gold enough: Repair now in the middle of a fight: no portal until the pack is dead; the Why line says waiting; loot is picked up; one portal, Charsi (or the act's smith) repairs, the log has `repair trip done before=.. after=..`, the hero returns through its own portal and continues.
3. Without a scroll: the hero walks home (or takes the waypoint) and back; in town: no portal, straight to the smith; with gold in the stash only: the withdraw line `town: withdrew gold for the repair`.
4. Make the smith fail (stand away from her with a panel open): one retry, the request stays queued with the reason, no second trip for 20 min.
5. Cancel while waiting and while farming; restart the bot in each phase; Stop farming on the repair farm.
6. Look at the arrival in the field: is the hero safe when a pack stands at the portal (the crowd re-check is not possible from town).
7. `d2cheat ledger` shows no writes from any of this.

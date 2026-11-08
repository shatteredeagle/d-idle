# Hero risk meter (TASK-159, unlockrisk)

Design note: `docs/research/unlockables.md` row 9 (a brainstorm). Built and tested offline on fixtures. Tags: V verified by the named offline test, U unverified.

## What it is

A gauge over the hero's own numbers that turns into one verdict before a boss: `ready`, `marginal`, `turn_back`. Code `internal/watch/unlockrisk.go`, table `internal/watch/risk.json`, ladder row `risk` (group ui).

## It cannot contradict the bot

The meter has no formula for the bot's decisions. It reads the bot's own answers through `RiskFacts` (the bot supplies them, like `ProposalFacts`):

| check | source | block / warn |
|---|---|---|
| guard | `GuardWhy`: the bot's `areaGuardWhy` for the zone (`internal/bot/routeguard.go`) | block when not empty |
| kit | `Kit`: `ladder.KitFor` (the proposer's and the tower guard's enum) | NONE blocks, THIN warns |
| potions | `Potions` against the proposer table's `minPotions` (`decide.ProposeTable`) | warn below |
| life | `LifePct` against `MinEngagePct` and `EngagePct` (resource tuning `MinEngageLifePct`, `EngageLifePct`) | block below the first, warn below the second |
| level | clvl against the zone minimum plus the class lead: the tally zone table (`currency.TallyRows.Zones`) and the proposer table's `classes` | warn below |
| resist | `ResistGap` (U: no zone resist table exists in the repo; 0 = unknown) | warn above 0 |
| close calls | the meter's own record, `closeCallsMarginal` (2) in `closeCallWindowMin` (30) minutes | warn at the count |

Any block gives `turn_back`, otherwise any warn gives `marginal`, else `ready`. The gauge is the share of checks that are ok, capped below 30 for turn back and below 70 for marginal. An unknown class or a zone without a range is an error naming it (no Sorceress fallback). V `TestRiskVerdictAgainstTheGuards`: for Sorceress and Paladin, over levels 8 to 22, merc, potions and portal, the verdict is `turn_back` exactly when `task.TowerWhy` or an empty kit refuses, and is never `ready` when the guard, the kit or the potion floor objects. V `TestRiskVerdictLifeLevelResistAndCloseCalls` (table), `TestRiskUnknownClassAndZoneAreErrors`.

## Unlock

First near-death survived: the life reading falls below `nearDeathPct` (25) and later recovers to `recoverPct` (50) or more. `RiskMeter.Tick` (called from `Fun.Settle`) records one close call and reports the viewer-side moment `near_death_survived` to the ladder (trigger kind `seen`, no new kind). A death while the dip is open drops it without a record (V `TestRiskUnlockByNearDeathGoldenTwoClasses`: a dip that ends in death, a 30 percent dip and a 45 percent partial recovery open nothing; reveal line once, a second close call reveals nothing again). Close calls are kept in memory only (a restart forgets them; the unlock itself is persisted by the ladder).

## API and return screen

- `GET /api/risk[?zone=Name]` (viewer): `watch.risk/1`, `open`, `verdict {verdict, gauge, zone, reasons[], checks[]}`, `closest`, `recentCloseCalls`. Locked: the teaser only. Without `zone` the bot picks its next target. Goldens `testdata/risk-sorceress.json`, `risk-paladin.json`.
- Return screen section `risk`, "Closest call": the lowest dip since the viewer left, hidden when there was none or the row is shut (V `TestRiskReturnSectionClosestCall`).

## Not done

- Wired in `cmd/d2bot` (TASK-185): `FunConfig.Risk` with `bot.Runner.RiskFacts` (`internal/bot/riskfacts.go`: `GuardWhy` from the route guard through `OrderGuardWhy`, `Kit` from `ladder.KitFor`, the engage lines from the resource tuning, life percent and `Dead` from the snapshot); `Fun.Settle` feeds `Tick` on the 2 s poll. V `TestRiskFacts` (bot), `TestRiskMeterWiredToTheBotFacts` (cmd/d2bot, two classes). `ResistGap` is 0 (U).
- A viewer panel is not built; the data is the route. Live: nothing run against a game; every threshold in `risk.json` is U.

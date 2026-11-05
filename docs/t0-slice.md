# T0 slice (TASK-21, phase A2: offline)

> status: phase A2 is code and tests only. The flag `ladder.enabled` is OFF by default. The live drill (phase B) has not run: `docs/probes/t0drill.md`. Tags: V = proven by a test in this tree, U = unverified, S = from a doc.

The first wiring of the decision-point store (`internal/decide`, `docs/research/decision-point-store-spec.md`) into the bot, under one rule: the store must not change behaviour until a person teaches it something. Phase A2 uses the public API Track 2 added: `Store.AskWithDefault(ctx, legacy)` makes the legacy code's own answer the untrained default, so parity is true by construction at every wired point; the golden grids stay as the proof (V: `go test -p 1 ./internal/ladder ./internal/town ./internal/bot`).

## What is wired

| point | call site | legacy answer passed as the default | parity grid (V) |
|---|---|---|---|
| `t1.repair` | `town.(*Town).repairDue` | `town.RepairDue(pct, repairBelowPct)` | 4848 states: 101 durabilities x 8 gold x 3 difficulties x 2 builds |
| `t1.tp.lost` | `town.tpBuyAllowed`, after `resource.BuyKit` allows the purchase | `buy` (BuyKit true) | 12006 states: gold 0..2000 x 3 area classes x 2 builds |
| `t1.return` | `bot.(*Runner).fieldVerdict` via `askReturn` | `resource.FieldGate` Go | 280 states: belt gate off and on (50 %), 0..6 potions, portal, 5 gold values, 2 builds; 85 are legacy refusals |
| `t1.town.trigger` | `bot.(*Runner).tripVerdict` via `askTrip` | `resource.TownTripTrigger` Go | 1080 states, 200 with a Go verdict: 9 potion counts x 6 gold x 5 pack fills x 2 difficulties x 2 builds |
| `t1.restock` | `town.(*Town).restockFree`, in `buyScrolls` where the trip gold floor limits spending | gold at or above `resource.tripGoldFloor` (the gold clause of `TripBudgetSpent`) | 12006 states: gold 0..2000 x 3 belt states x 2 builds |

Each grid asserts ladder answer == the legacy function called directly, and also counts where the catalogue default (which a call site no longer uses) differs from the legacy answer: 0 for `t1.repair`, `t1.tp.lost`, `t1.restock`; 24 of the 200 Go verdicts of `t1.town.trigger` (the policy trips for mana or free-heal reasons the key does not carry); 85 of 280 for `t1.return` (every legacy refusal: the catalogue always says return). Where they differ the legacy answer wins; that is the point of `AskWithDefault`.

`TestLearnedRuleOverridesLegacy` (V): for each of the 5 points, a scripted player teaches the opposite answer, the rule is learned (two answers, confirm yes) and replaces the legacy answer, for 2 builds (Sorceress, Paladin); other contexts are untouched.

### Seams (what a learned answer can do)

The ladder only chooses between actions the legacy code could also reach; legacy guards stay.
- `t1.repair`: learned `skip` stops a repair the knob wants; learned `repair` starts one (latch and gold guards still apply).
- `t1.tp.lost`: asked only after `BuyKit` allowed the purchase; learned `walk` vetoes it. `buy` cannot create a purchase the gold does not allow.
- `t1.return`: learned `no` turns a Go into a restock round (`FieldVerdict.Restock`), bounded by `gateRestocks`; `yes` never lifts a refusal (a poor character still goes to its first field only). The key's area class is "field" (where the hero goes), kit OK/THIN/NONE is `bot.askReturn`'s mapping (NONE: no portal and no healing potion; THIN: no portal or under 2; U, simplified from the SME review which also counts gold).
- `t1.town.trigger`: asked only when the policy says go; learned `continue` stops the trip, `go` cannot invent one (no method). Belt band: `ladder.BeltBand` (full at plan capacity, low under half; U, the half/low edge is my choice), pack fill from `beltAndInv`.
- `t1.restock`: the only legacy site that uses the trip gold floor in town is the scroll cap in `buyScrolls`. Learned `no` caps scrolls at 1 so the gold goes to potions; learned `yes` removes the cap below the floor. This is the least direct mapping: `TripBudgetSpent` itself is used inside `resource.thinBeltRetreat` (a fight-time decision in the owner-restricted `internal/resource`), which is not touched (U: whether the town site is where a person means "restock").

## Waiting for Track 2b (not wired)

`t0.retreat.drink`, `t0.area.toohard`, `t0.unique.fight`, `t2.mercenary` still have band-interior splits in `TestDefaultParity` (champion flag, merc alive, finer life and gold edges). `AskWithDefault` would make them parity-true at a call site, but the call sites live in `internal/resource` and the fight loop (a decision per tick), and the contexts need fields the hero mapping cannot fill yet. Not wired before 2b adds the context fields. No T0 point is wired, so the card's T0 acceptance (items 1 and 2) is NOT met; the drill is a town drill.

Not wired, no witness test (U, code read): `t0.retreat.repeat` (no legacy per-area retreat counter), `t1.identify`, `t1.keep.kind`, `t2.failstreak`, `t2.reset.game`.

## How it works

- `internal/ladder`: `Ladder.Decide(point, hero, extra, legacy)` builds the context key (`decide.Catalogue.Build`: banded fields, class from the catalogue class table through `d2.Class.String`, an unknown class is an error naming it), calls `Store.AskWithDefault` (learned rule, else the legacy answer). Errors make the call site keep the legacy answer and count a fault. Hardcore is always `normal` (the bot does not read it yet, U).
- Thoughts: the ladder subscribes to the store (`Subscribe`); `Event.Info` (a `ThoughtInfo`) fills its queue, no polling of `OpenThoughts` except once to pick up thoughts a persisted store reopened. A `Sink` shows them (`LogSink` writes `ladder: thought <point> key=<key> choices=a|b default=<legacy answer>`); an `AnswerSource` (`ScriptFile`/`FileSource`, `ProfileSource`) replies after a virtual delay; nothing reads stdin or the network. The answer profiles are `decide.KnownProfiles/Holdout/NewProfile` (the ladder keeps no copy; `ProfileSource` is a thin adapter). Nothing waits: the decision is returned first, the question is put at the next `Pump` outside a fight.
- `bot.(*Runner).EnableLadder(dir, profile, answers)`: returns nil and installs nothing when `ladder.enabled` is 0. `New` does not call it; cmd/d2bot wiring is not done. With nothing installed `ladder.Active()` is nil and each call site runs the legacy line only (V: `TestLadderCallSites`, `TestEnableLadderFollowsTheKnob`, `TestLadderTripAndReturnPoints`).

## Tests

- Golden parity: the five grids above plus `TestRefusedPointsNowAgree` (witnesses of the points that stay unwired).
- Profiles: 5 answer profiles (cautious, bold, inconsistent, 2 seeded holdouts) x 2 builds, ten virtual minutes of random calls to all five points (`TestProfilesKeepTheInvariants`); 30 seeded random answer scripts (`TestRandomScriptsNeverStall`); invariants: every call returns, no error, the log stays under 4 lines per call, no rule revoked, open thoughts bounded.
- Call sites: `internal/town/ladder_test.go` (repair, scroll, restock with the ladder off and on), `internal/bot/thoughts_test.go` (trip and field verdicts equal the legacy ones with the ladder on; a learned answer stops the trip or the departure). The bot rig is the default Sorceress; the two-build proof is in the ladder grids.
- Drill: `internal/drill/t0_test.go` runs the T0 group on a fake lab for 5 profiles x 2 builds; a CHEATED run fails; the `file` profile and bad inputs block.
- `ladder.enabled = 0`: the combat, bot, town, resource and env package tests run unchanged.

## T0 drill (phase B, not run)

`cmd/d2drill -group T0`: setup (empty pack, gold 1000, worn gear to durability 1) before `d2cheat runstart`, then 10 virtual minutes of town passes (repair and scrolls chores) through the bot's real town code with the ladder installed. Exercised by the live drill: `t1.repair`, `t1.tp.lost`, `t1.restock` (the town chores). `t1.return` and `t1.town.trigger` live in the bot runner's trip logic, which the town lab does not drive: they are proven offline only (U live). Exact commands and what is asserted: `docs/probes/t0drill.md`.

## API needs for Track 2b

1. A way to ask whether a point's catalogue default and a legacy answer agree without a store (`Catalogue.Default` is enough; used as `Ladder.CatalogueDefault`). No change needed.
2. For the T0 points: the fields `champion` (pack flag), `merc` alive, and finer `life`/`gold` edges, as named in the notes; the hero mapping will fill them from the snapshot when they exist.
3. `ThoughtInfo` has no `OpenedAt`; the ladder uses its own clock at the event (and `Expires - FadeSec` for reopened thoughts). Adding `Opened` would remove that.
4. `Obs` has no hardcore flag source (the bot does not read hardcore), so the key carries `normal`.

## Not done

Blood Moor field drill, any T0 wiring, a call to `EnableLadder` from cmd/d2bot, the watch-screen thought UI (Track 3), the live drill (phase B).

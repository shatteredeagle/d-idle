# T0 slice (TASK-21, phase A3: offline)

> status: phase A3 is code and tests only (A3 wires the four T0 fight-time points on top of the five T1 points of A2). The flag `ladder.enabled` is OFF by default. The live drill (phase B) has not run: `docs/probes/t0drill.md`. Tags: V = proven by a test in this tree, U = unverified, S = from a doc.

The first wiring of the decision-point store (`internal/decide`, `docs/research/decision-point-store-spec.md`) into the bot, under one rule: the store must not change behaviour until a person teaches it something. Phase A2 uses the public API Track 2 added: `Store.AskWithDefault(ctx, legacy)` makes the legacy code's own answer the untrained default, so parity is true by construction at every wired point; the golden grids stay as the proof (V: `go test -p 1 ./internal/ladder ./internal/town ./internal/bot`).

## What is wired

| point | call site | legacy answer passed as the default | parity grid (V) |
|---|---|---|---|
| `t1.repair` | `town.(*Town).repairDue` | `town.RepairDue(pct, repairBelowPct)` | 4848 states: 101 durabilities x 8 gold x 3 difficulties x 2 builds |
| `t1.tp.lost` | `town.tpBuyAllowed`, after `resource.BuyKit` allows the purchase | `buy` (BuyKit true) | 12006 states: gold 0..2000 x 3 area classes x 2 builds |
| `t1.return` | `bot.(*Runner).fieldVerdict` via `askReturn` | `resource.FieldGate` Go | 280 states: belt gate off and on (50 %), 0..6 potions, portal, 5 gold values, 2 builds; 85 are legacy refusals |
| `t1.town.trigger` | `bot.(*Runner).tripVerdict` via `askTrip` | `resource.TownTripTrigger` Go | 1080 states, 200 with a Go verdict: 9 potion counts x 6 gold x 5 pack fills x 2 difficulties x 2 builds |
| `t1.restock` | `town.(*Town).restockFree`, in `buyScrolls` where the trip gold floor limits spending | gold at or above `resource.tripGoldFloor` (the gold clause of `TripBudgetSpent`) | 12006 states: gold 0..2000 x 3 belt states x 2 builds |
| `t0.retreat.drink` | `combat.(*Fighter).askDrink`, in `abortFight` after the survival step found nothing to drink; asked only below 65 percent life | the `resource.AbortFight` kind mapped to an answer (continue fight, portal, leave, flee or retreat run) | 19200 states: 2 tunings (shipped, RetreatBelt 0) x 2 builds x Normal/Nightmare x softcore/hardcore x life 1..100 x held none/hp/rv x portal x champion pack; the oracle is `PotionPolicy` then `AbortFight`, the legacy answer `drink` included |
| `t0.unique.fight` | `combat.(*Fighter).askUnique`, in `preEngage`, with an elite among the candidates and life at or under 70 percent | avoid when `PreEngage` says Avoid or life is under `resource.minEngageLifePct`, else fight | 9600 states: 2 builds x Normal/Nightmare x life 1..100 x portal x 3 potion counts x pack danger 0..3 |
| `t0.area.toohard` | `bot.(*Runner).towerWhy` (from `fillRoute` and `towerGuardWhy`), only while the tower guard is on (Normal) | `task.TowerLevelWhy` open: go; closed with kit NONE: lower; else skip | 1440 states: 2 builds x Normal/Nightmare x clvl 1..40 x merc alive/dead/none x 3 kits |
| `t2.mercenary` | `bot.(*Runner).askMerc`, from `mercWanted` after the merc lock is released, only where the legacy gate already says hire or revive | true (hire) at that point; the grid's legacy is `merc != alive && gold >= bot.mercHireMinGold` | 54018 states: 2 builds x merc alive/dead/none x 3 kits x gold 0..3000 |

Each grid asserts ladder answer == the legacy function called directly, and also counts where the catalogue default (which a call site no longer uses) differs from the legacy answer: 0 for `t1.repair`, `t1.tp.lost`, `t1.restock`; 24 of the 200 Go verdicts of `t1.town.trigger` (the policy trips for mana or free-heal reasons the key does not carry); 85 of 280 for `t1.return` (every legacy refusal: the catalogue always says return). Where they differ the legacy answer wins; that is the point of `AskWithDefault`. T0 grids: 0 states where the catalogue default differs for `t0.retreat.drink`, `t0.unique.fight` and `t2.mercenary`; `t0.area.toohard` differs only in Nightmare (the guard is off there, the catalogue is Normal-only, and the call site does not ask when the guard is off). V: `go test -p 1 ./internal/ladder` (`TestParityRetreatDrink`, `TestParityUniqueFight`, `TestParityTooHard`, `TestParityMercenary`).

`TestLearnedRuleOverridesLegacy` (V): for each of the 5 T1 points, a scripted player teaches the opposite answer, the rule is learned (two answers, confirm yes) and replaces the legacy answer, for 2 builds (Sorceress, Paladin); other contexts are untouched. `TestLearnedRuleOverridesLegacyT0` does the same for the four T0 points (2 builds each); call-site tests: `internal/combat/ladder_t0_test.go` (a taught `leave` replaces the break-away with the chicken, a taught `avoid` skips an elite pack), `internal/bot/ladder_t0_test.go` (a taught `go` lifts the tower level block, a taught `skip` blocks an open one, a taught `wait` postpones the hire).

### Seams (what a learned answer can do)

The ladder only chooses between actions the legacy code could also reach; legacy guards stay.
- `t1.repair`: learned `skip` stops a repair the knob wants; learned `repair` starts one (latch and gold guards still apply).
- `t1.tp.lost`: asked only after `BuyKit` allowed the purchase; learned `walk` vetoes it. `buy` cannot create a purchase the gold does not allow.
- `t1.return`: learned `no` turns a Go into a restock round (`FieldVerdict.Restock`), bounded by `gateRestocks`; `yes` never lifts a refusal (a poor character still goes to its first field only). The key's area class is "field" (where the hero goes), kit OK/THIN/NONE is `bot.askReturn`'s mapping (NONE: no portal and no healing potion; THIN: no portal or under 2; U, simplified from the SME review which also counts gold).
- `t1.town.trigger`: asked only when the policy says go; learned `continue` stops the trip, `go` cannot invent one (no method). Belt band: `ladder.BeltBand` (full at plan capacity, low under half; U, the half/low edge is my choice), pack fill from `beltAndInv`.
- `t1.restock`: the only legacy site that uses the trip gold floor in town is the scroll cap in `buyScrolls`. Learned `no` caps scrolls at 1 so the gold goes to potions; learned `yes` removes the cap below the floor. This is the least direct mapping: `TripBudgetSpent` itself is used inside `resource.thinBeltRetreat` (a fight-time decision in the owner-restricted `internal/resource`), which is not touched (U: whether the town site is where a person means "restock").
- `t0.retreat.drink`: the legacy drink decisions (`combat.Decide`, the survival step) run before and are not replaced: a learned `fight` cannot suppress a potion the survival step already chose. At the abort step a learned `fight` cancels the escalation, `portal`, `leave` and `run` replace it (a portal that cannot open falls through like the legacy one), `drink` tries one healing potion and otherwise keeps the legacy verdict. Key mapping: life percent, `held` hp over rv from the belt (`ladder.HeldFor`), kit from `ladder.KitFor` (NONE no portal and no healing potion, THIN no portal or under 2; U, not the decide oracle's "NONE = no portal"), `champbonus early` from `resource.ChampionEarly`. Hardcore is `Hero.Hardcore`, which the fighter does not know: it asks as softcore (U).
- `t0.unique.fight`: asked per pre-engage tick with an elite candidate (unique, champion, minion) at life 70 percent or less; deduplicated by key in the store. Learned `avoid` ends the approach like the legacy Avoid; learned `fight` proceeds past the legacy rest, drink or avoid, so a person who says "fight at 30 percent" is believed. Rest and drink before a pack stay when the answer equals the legacy one. U: the legacy answer "avoid under minEngageLifePct" is the decide oracle's, the real code heals first there.
- `t0.area.toohard`: the only wired area is the Forgotten Tower guard (the tall dungeon floor, the oracle's). A learned `go` lifts the level block and keeps the kit part of the guard; `skip` and `lower` both block (there is no lower-area detour in the bot: U, `lower` equals `skip` in effect). The Monastery road floor is not asked. Asked from `fillRoute`, so the thought may open before the hero nears the tower (U). `Hero` area class is "field" (where it goes); `merc` is `task.MercState`.
- `t2.mercenary`: only a veto: a learned `wait` postpones a hire or revive the legacy gate wanted (the chore asks again after its pause); a learned `hire` cannot create one the town, the gold or the legacy `known` rule do not allow. The revive path asks with merc `dead` although its legacy gate is the gold floor, not 400 (U, weak mapping).

## T0 points (A3)

Track 2b and 2c closed every default-parity split (0 of 23 points; `docs/research/decision-point-store-notes.md`), so the four T0 points are wired in A3 with the legacy verdict as the untrained answer. Each call site is guarded by `ladder.Active() != nil`; with the ladder off (`ladder.enabled` 0, the default, and nobody calls `EnableLadder`) each site runs its legacy line only and the existing combat, bot, resource, task and town tests run unchanged (V). Refused: none; no wired point shows a mismatch. Weak mappings (U): the kit classes, hardcore (fighter), `lower` equal to `skip`, the unique-fight legacy answer, the revive path of `t2.mercenary`, the tower thought timing.

Not wired, no witness test (U, code read): `t0.retreat.repeat` (no legacy per-area retreat counter), `t1.identify`, `t1.keep.kind`, `t2.failstreak`, `t2.reset.game`.

## How it works

- `internal/ladder`: `Ladder.Decide(point, hero, extra, legacy)` builds the context key (`decide.Catalogue.Build`: banded fields, class from the catalogue class table through `d2.Class.String`, an unknown class is an error naming it), calls `Store.AskWithDefault` (learned rule, else the legacy answer). Errors make the call site keep the legacy answer and count a fault. Hardcore is always `normal` (the bot does not read it yet, U).
- Thoughts: the ladder subscribes to the store (`Subscribe`); `Event.Info` (a `ThoughtInfo`) fills its queue, no polling of `OpenThoughts` except once to pick up thoughts a persisted store reopened. A `Sink` shows them (`LogSink` writes `ladder: thought <point> key=<key> choices=a|b default=<legacy answer>`); an `AnswerSource` (`ScriptFile`/`FileSource`, `ProfileSource`) replies after a virtual delay; nothing reads stdin or the network. The answer profiles are `decide.KnownProfiles/Holdout/NewProfile` (the ladder keeps no copy; `ProfileSource` is a thin adapter). Nothing waits: the decision is returned first, the question is put at the next `Pump` outside a fight. "Fight" means an engagement (`Runner.engaged`: a living hostile within `combat.fightRadius` of the last snapshot, the radius the safety rules use), not the clearing state: walking a field with nothing near is not a fight, so a question reaches the player there and is held only while engaged (live drill 2026-11-05, defect 1).
- `bot.(*Runner).EnableLadder(dir, profile, answers)`: returns nil and installs nothing when `ladder.enabled` is 0. `New` does not call it; cmd/d2bot wiring is not done. With nothing installed `ladder.Active()` is nil and each call site runs the legacy line only (V: `TestLadderCallSites`, `TestEnableLadderFollowsTheKnob`, `TestLadderTripAndReturnPoints`).

## Tests

- Golden parity: the five grids above plus `TestRefusedPointsNowAgree` (witnesses of the points that stay unwired).
- Profiles: 5 answer profiles (cautious, bold, inconsistent, 2 seeded holdouts) x 2 builds, ten virtual minutes of random calls to all five points (`TestProfilesKeepTheInvariants`); 30 seeded random answer scripts (`TestRandomScriptsNeverStall`); invariants: every call returns, no error, the log stays under 4 lines per call, no rule revoked, open thoughts bounded.
- Call sites: `internal/town/ladder_test.go` (repair, scroll, restock with the ladder off and on), `internal/bot/thoughts_test.go` (trip and field verdicts equal the legacy ones with the ladder on; a learned answer stops the trip or the departure). The bot rig is the default Sorceress; the two-build proof is in the ladder grids.
- Drill: `internal/drill/t0_test.go` runs the T0 group on a fake lab for 5 profiles x 2 builds; a CHEATED run fails; the `file` profile and bad inputs block.
- `ladder.enabled = 0`: the combat, bot, town, resource and env package tests run unchanged.

## How to run it (integrated: d2bot -ladder -watch)

Both switches are OFF by default; without `-ladder` nothing in `cmd/d2bot/ladder.go` runs: no directory is made, no port is opened (V: `TestLadderOffDoesNothing`, `TestLadderFlagRules`). `-ladder` sets the knob `ladder.enabled` 1 for this run only, then `bot.Runner.EnableLadderWith` builds the store, hands it to the watch Hub and installs the ladder (before the runner starts). Works with `--attach` and `--managed` (live and d2d backend); refused with `--sim` (the goals simulator has no bot runner) and `--dry`.

| flag (env) | meaning |
|---|---|
| `-ladder` (`D2BOT_LADDER`) | ladder on; store under `<data>/instances/<id>/state/ladder/decide-<profile>.json` |
| `-ladder-store NAME` (`D2BOT_LADDER_STORE`) | store directory `state/ladder/runs/NAME`. Default: the run named by `D2BOT_RUN` (drills are independent); no run named: the shared `state/ladder` store, so real play keeps learning across sessions; `shared` forces the shared store (continue a learned profile in a named run) |
| `-ladder-profile NAME` or `holdout:SEED` (`D2BOT_LADDER_PROFILE`) | scripted player for drills: cautious, bold, inconsistent, holdout:SEED. Without it a person answers on the watch screen |
| `-watch` (`D2BOT_WATCH`) | serve the player screen from the bot; needs `-ladder` |
| `-watch-addr` (default `127.0.0.1:8891`) | read-only viewer (GET routes only) |
| `-watch-answer-addr` (default `127.0.0.1:8892`) | SEPARATE listener for answer, confirm, revoke, orders |
| `-watch-origin` | comma list of page origins the answer route accepts (default: the viewer's port on localhost) |
| `-token` | required as soon as either address is not loopback (bearer header, `?t=` once, or cookie) |

`-ladder` alone needs someone to answer: add `-watch` or `-ladder-profile`. `-ladder-profile` with `-watch` lets the script answer while the screen shows everything (drills).

Local, a person answers:
```
d2bot --attach --ladder --watch --goals 'farm:Blood Moor'      # then open http://127.0.0.1:8891/watch
```
LAN, with a token (the page is reached as `http://HOST:8891/watch?t=TOKEN`; the answer route is on 8892):
```
d2bot --attach --ladder --watch --watch-addr 0.0.0.0:8891 --watch-answer-addr 0.0.0.0:8892 \
      --watch-origin http://HOST:8891 --token "$(cat ~/.d2bot-token)" --goals 'farm:Blood Moor'
```
Scripted drill (no person, screen optional):
```
d2bot --attach --ladder --ladder-profile holdout:101 --watch --goals 'farm:Blood Moor' --run-for 10m
```
What it does with the screen: thoughts of the nine wired points reach the Hub through the ladder `Sink`; an answer posted on the answer route goes straight to the store; an order becomes a `farm_area` goal (area, `UntilMins`) at the front of the goal manager and cancelling it removes the goal ("fight here" uses the area of the runner's last snapshot, "come home" an idle goal, U: no go-to-town goal type). Uptime comes from the goal manager's events: death = death, the first area change after it = recovery, the next kill = progress, an error event or runner status error = stall (`watch.GoalsOrders.OnGoalEvent/OnStatus`, V: `TestUptimeFromGoalEvents`); a town stall has no source yet (U).

`writes after start: 0`: the ladder and the screen write no game memory; the only writes are the store file and the bot's own input. A measured run still declares its setup before `d2cheat runstart` as always; the run record takes `writes after start` from `d2cheat ledger --since-run <run>`.

Tested offline only (V): `go test -p 1 -count=1 ./cmd/d2bot ./internal/watch ./internal/bot -run 'Ladder|Uptime|Serve|Token|Origins'` (fake enable on a virtual clock, two builds, human and scripted answers, orders to the real goal manager, viewer read-only, token rule). U live: nothing here has met the real game.

## T0 drill (phase B, not run)

`cmd/d2drill -group T0`: setup (empty pack, gold 1000, worn gear to durability 1) before `d2cheat runstart`, then 10 virtual minutes of town passes (repair and scrolls chores) through the bot's real town code with the ladder installed. Exercised by the live drill: `t1.repair`, `t1.tp.lost`, `t1.restock` (the town chores). `t1.return` and `t1.town.trigger` live in the bot runner's trip logic, which the town lab does not drive: they are proven offline only (U live). Exact commands and what is asserted: `docs/probes/t0drill.md`.

## API needs for Track 2b

1. A way to ask whether a point's catalogue default and a legacy answer agree without a store (`Catalogue.Default` is enough; used as `Ladder.CatalogueDefault`). No change needed.
2. Done in 2b/2c: the T0 fields exist and the A3 sites fill them (`life`, `held`, `champbonus`, `packdanger`, `merc`, `clvl`, `kit`, `gold`).
3. Done: `ThoughtInfo.Opened` exists and the ladder uses it (A3 removed its own clock).
4. `Hero.Hardcore` exists; no call site reads hardcore from the game yet, so every site passes softcore (U).

## Not done

Blood Moor field drill (live), a live lab hook for the fight-time points, the live drill (phase B). Done since: the d2bot call (`-ladder`, see "How to run it") and the watch screen.

## T0 fight-time points in the drill (A3)

`drill.T0FightLab` is an optional interface of the lab (`FightPoints(ctx, point)`); when the lab implements it the drill asks each of the four fight-time points once per round next to the chores and counts the passes (`fightPasses` in the run record; 80 for 20 rounds). The offline fake lab (`internal/drill/t0_test.go`, `TestT0DrillRunsTheFightPoints`: 5 profiles x 2 builds, 10 virtual minutes) implements it by asking the ladder with states that move with the pass count. The live lab (`cmd/d2drill`) does not implement it yet, so a live run exercises the town points only; what it needs is in `docs/probes/t0drill.md` ("Live run needs"). `cmd/d2bot -ladder` now calls `bot.EnableLadderWith` (integration, "How to run it"); off by default.

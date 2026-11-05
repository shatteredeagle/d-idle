# T0 slice (TASK-21, phase A: offline)

> status: phase A is code and tests only. The flag `ladder.enabled` is OFF by default. The live drill (phase B) has not run: `docs/probes/t0drill.md`. Tags: V = proven by a test in this tree, U = unverified, S = from a doc.

The first wiring of the decision-point store (`internal/decide`, `docs/research/decision-point-store-spec.md`) into the bot, under one rule: the store must not change behaviour until it reproduces today's behaviour.

## What is wired (V: `go test -p 1 ./internal/ladder ./internal/town`)

| point | tier | call site | legacy test it must equal | parity grid |
|---|---|---|---|---|
| `t1.repair` | 1 | `town.(*Town).repairDue`, used by `needRepair` | `town.RepairDue(pct, town.repairBelowPct)` | 4848 states: 101 durabilities x 8 gold values x 3 difficulties x 2 builds |
| `t1.tp.lost` | 1 | `town.tpBuyAllowed`, in `buyFirstTP` after `resource.BuyKit` | `resource.BuyKit(..., KitTPGold 100).TP` | 12006 states: gold 0..2000 x 3 area classes x 2 builds |

Neither is a T0 point. The brief asked for the T0 points the default rule file covers fully; the notes table lists only `t0.retreat.repeat` (1 of 4, under the three needed), so the covered T1 points were wired too. Of the nine fully covered points these two are the only ones with a pure legacy test to compare with.

Seam: the ladder can only choose between the answers the legacy code could also reach. The legacy guards stay (the repair latch, gold for the estimate, gold against the real scroll price); a ladder answer "walk" can veto a scroll purchase, "buy" cannot create one the gold does not allow. With the ladder on, `town.repairBelowPct` no longer decides (the band edges 20/30 of the catalogue do): at the default knob 30 the two are equal (V, grid above).

## Refused by the parity test (stay on the legacy path, untouched)

`TestRefusedPointsDisagree` keeps one witness state per refusal and fails if the disagreement ever disappears (then the point may be wired).

| point | witness (V, test) |
|---|---|
| `t0.retreat.drink` | life 30 %, kit OK, potion held: legacy `PotionPolicy` drinks, the rule file says portal (25-35 band); at 90 % the legacy drinks nothing, the file says drink (no "keep fighting" answer) |
| `t1.return` | legacy `FieldGate` (knob 0) always lets the hero leave town, the file keeps a kit-NONE hero in town |
| `t1.town.trigger` | legacy `TownTripTrigger` does not trip with a full belt, the file says go at 0 free rows |
| `t1.restock` | legacy `TripBudgetSpent` refuses under tripGoldFloor 300, the file restocks from 100 gold |

Not wired without a test witness (U, code read): `t0.retreat.repeat` (no legacy per-area retreat counter; `brAttempt` waits for a level, which the key does not carry), `t0.area.toohard` (route floors are absolute levels, the key has a relative gap), `t0.unique.fight` (no legacy "avoid a champion pack"), `t1.identify` (`idScrollWanted` buys a scroll for a likely valuable with none held, the file says leave), `t1.keep.kind`, `t2.failstreak` (gearDeathsHigh 4 vs the file's 3), `t2.reset.game`. The ten uncovered points have no rule-file row at all.

Consequence: no T0 point can be wired yet. The T0 drill of the card (Blood Moor, 10 virtual minutes) would exercise nothing until the rule file or the catalogue changes (see "API changes wanted").

## How it works

- `internal/ladder`: `Ladder.Decide(point, hero, extra)` builds the context key (`decide.Catalogue.Build`: banded fields, class from the catalogue class table through `d2.Class.String`, an unknown class is an error naming it), asks the store (learned rule), else the shipped rule file, else the catalogue default. Errors make the call site fall back to the legacy answer and count a fault. Hardcore is always `normal` (the bot does not read it yet, U).
- Thoughts: a thought opened by the store is queued and shown to a `Sink` (`LogSink` writes `ladder: thought <point> key=<key> choices=a|b default=a` to the bot log); the store keeps the answer log. An `AnswerSource` (`ScriptFile`/`FileSource`, `ProfileSource`) replies after a virtual delay; nothing reads stdin or the network. Nothing waits: the decision is returned first, the question is put at the next `Pump` outside a fight (`bot.EnableLadder` reports the clearing state as a fight, U).
- Virtual time: the store's fade clock is injected (`ClockFunc`; the bot passes its own clock counted from the enable call, the drill passes game ticks).
- `bot.(*Runner).EnableLadder(dir, profile, answers)`: returns nil and installs nothing when `ladder.enabled` is 0. `New` does not call it; a caller opts in (cmd/d2bot wiring is not done, see below). With nothing installed `ladder.Active()` is nil and each call site runs the legacy line only (V: `TestLadderCallSites`, `TestEnableLadderFollowsTheKnob`).

## Tests

- Golden parity: the two grids above, plus the refusal witnesses (`internal/ladder/parity_test.go`).
- Profiles: 5 answer profiles (cautious, bold, inconsistent, 2 seeded holdouts) x 2 builds (Sorceress, Paladin), ten virtual minutes each (`TestProfilesKeepTheInvariants`); 30 seeded random answer scripts with arbitrary replies, delays and confirms (`TestRandomScriptsNeverStall`); invariants: every call returns, no error, the log stays under 4 lines per call, no rule revoked, open thoughts bounded.
- Scripted rule: two answers at one key learn a rule that changes the next decision (`TestScriptedAnswersLearnARule`); no question during a fight; a thought fades on virtual time; an unknown class is a counted fault.
- Drill: `internal/drill/t0_test.go` runs the T0 group on a fake lab for 5 profiles x 2 builds, a CHEATED run (writes after start) fails, the `file` profile and bad inputs block.
- `ladder.enabled = 0`: the combat, bot, town, resource and env package tests run unchanged (see the report for the commands and results).

## T0 drill (phase B, not run)

`cmd/d2drill -group T0`: setup (empty pack, gold 1000, worn gear to durability 1) before `d2cheat runstart`, then 10 virtual minutes of town passes (repair, scrolls chores) through the bot's real code with the ladder installed; the record carries the rule-file hash, rules learned, answers given, thoughts faded and `writes after start: N` (a non-zero N fails the drill as CHEATED). Because only town points are wired, the drill is a town drill, not the Blood Moor field drill the card names. Commands: `docs/probes/t0drill.md`.

## API changes wanted from internal/decide (not made; Track 2 owns it)

1. Export the answer profiles of `profiles_test.go` (known three, `holdout(seed)`) as a non-test file so the drill and the tests share one generator; the ladder carries a copy (`ladder.ProfileSource`) that mirrors them.
2. A public `Store.OpenThought(id)` or an event that carries the Thought, so a subscriber need not poll `OpenThoughts()`.
3. A catalogue entry per legacy-equal default for the T0 points, or an "adapter default" hook (`Store.AskWithDefault(ctx, legacyAnswer)`): the store answers a learned rule, else the caller's legacy action. That would let the T0 points be wired with parity by construction, and the rule file would stay a description of the expected defaults. Without it the T0 points need rule-file rows that match the knob code (no "keep fighting" answer exists above 55 % life).
4. `Obs` has no hardcore flag source; the bot does not read hardcore, so the key carries `normal`.

## Not done

Blood Moor field drill, any T0 wiring, a call to `EnableLadder` from cmd/d2bot, the watch-screen thought UI (Track 3), the live drill (phase B).

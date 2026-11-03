# Code review: the Gate 1 belt fixes (BELTFIX to BELTFIX5, HIREFIX)

Card: codereview1. Scope: `git log --oneline cd4c7cba..origin/main -- internal/bot internal/town internal/resource internal/d2dmerc internal/d2dsrc cmd/d2bot`
plus ce5e7f1e and 20cd22ae (BELTFIX, inside cd4c7cba). Reviewed at a45b9af4. Offline code reading only: no game, no logs
of the five Gate 1 runs (the minute files are not readable here), so every cost below is a bound taken from the code
(loop counts, timeouts, knob defaults in docs/knobs.md), not a measurement. Tags: V = read in the code at the line cited
(the behaviour follows from the code path, not run), S = follows from a knob default or a doc, U = not checked.

## Summary

The fixes do what their cards claim for the replayed windows, but three of them changed shared inputs that other
policies read, and those side effects are not tested:

1. `walkableHome` (BELTFIX3) is not only the "walk home" list. It is `Input.NearTown` for the trip policy, the fight
   abort and pre-engage. Adding 4, 5, 6, 9, 10, 17 turns "town too far" into "walk the level chain" for ordinary
   potion-low trips, and turns some "leave the game" decisions into "step back" in those six areas.
2. BELTFIX5 adds stash gold to `Input.Gold` but not to `LastTripGold`, so the futile-trip check can never fire while the
   stash holds 100 gold or more and no potion is carried.
3. The mana reserve (BELTFIX) holds back mana even when the kit is already complete. That fits the falling XP/h of a
   mana-bound Sorceress, but only as a hypothesis.

## Findings (most severe first)

### F1 (high) NearTown widening reaches the fight abort, pre-engage and every potion-low trip

`internal/resource/town.go:20` (V). `NearTownArea` is read by `bot.resInput` (`internal/bot/resource.go:21`) and by
`combat.nearTown` (`internal/combat/resource.go:29`). Three effects in Stony Field, Dark Wood, Black Marsh, Cave 1,
Underground Passage 1 and Burial Grounds when the character holds no portal source:

- Trip policy (V `internal/resource/town.go:101`). `TownTripTrigger` used to answer "no portal, town too far". It now
  returns `Go` by walk whenever healing potions are at or below `tripHPMin` (1). This is not limited to the 0-potion
  emergency of the card. Black Marsh to camp is five level changes
  (6 -> 5 -> 10 -> 4 -> 3 -> 2 -> 1, U: route from the nav graph not re-checked). Each attempt is bounded only by
  `townTimeoutSec` (180 s): `ToTown`, then `walkChainToTown` (`internal/town/trip.go:170`). The way back
  (`BackToField`, `travelTimeoutSec` 300 s) uses the waypoint when it is active, else walks the chain again. Worst case
  per trip: about 8 virtual minutes (180 + 300 s), against 0 before (no trip; the character fought on). The cards do
  not discuss the trade between "fight on at 1 potion" and "spend up to 8 minutes walking".
- Fight abort (V `internal/resource/abort.go:73`). `escape()` now returns `RetreatOut` (one kite step away, no time
  bound; `fleeMax` applies only to `Flee`, `internal/combat/resource.go:247`) where it returned `LeaveGame`. This
  covers "life and mana gone", "no potions, no mana, no melee" and "merc dead, hostiles remain". Scenario: clvl 11
  Sorceress in Black Marsh, 0 potions, no scroll, mana 3 and below the cheapest cast, life 40 %. Before, she chickened
  in about 1.8 s; now she steps back every tick until the chicken line (`lp <= chicken`) ends it. That costs the same
  game, later, and with the 0-potion death risk on the way. The chicken line still holds, so this is a delay with more
  damage taken, not a certain death (U live).
- Pre-engage (V `internal/resource/preengage.go:70`). With life under `engageLifePct` and regen too slow, the case
  `HasTP || NearTown` now gives `TownHeal` instead of `Avoid`. `TownHeal` calls `f.Act(TownPortal)`, which has no
  scroll and fails, and `st.skipPre` is set for 60 s (`internal/combat/resource.go:224`). For 60 s no pre-engage check
  runs, the `Avoid` of a low-life character included, so she engages at low life.

Smallest fix: keep `walkableHome` as it was (2, 8, 3) for `NearTown`. Add a second set, for example `chainHome`, read
only by `kitTrip`'s way-home test and `walkToTown`, so that only the 0-healing emergency trip may walk the chain.
Otherwise make `combat.nearTown` return the old three areas. Test: AbortFight in area 5 with no TP is `LeaveGame` for
"life and mana gone", and TownTripTrigger in area 6 with 1 potion and no TP is not `Go`.

### F2 (high) Stash gold defeats the futile-trip guard

`internal/bot/resource.go:71` against `:27` and `noteTripGold` (`:86`) (V). With no healing potion, `in.Gold` =
purse + stash, but `in.LastTripGold` is the purse after the last trip. The guard `in.Gold - in.LastTripGold <
FutileStep` (`internal/resource/town.go:86`) sees the stash as fresh income.

Scenario: purse 40, stash 1418, 0 potions, and the visit cannot turn gold into a potion. This happens when the
withdrawal fails (`MoveGold` not confirmed within `uiTimeout`), on any backend whose input is not a `goldMover` (only
`d2dtown.hoverBelt` implements it, `internal/d2dtown/adapter.go:268`), or when the vendor read fails. Then the
"healing potions low" trip is never futile. Every `resupplyCooldownSec/4` = 60 s the field trip repeats: walk or
portal home, a chore round of roughly 20 to 40 s (U), back out. Before BELTFIX5 the second trip was futile (40 - 40 <
100). The same `in.Gold` also switches off `Poor` and `PovertyCheck` (`internal/bot/leveling.go:135`) while the stash
cannot actually be reached.

Smallest fix: record `LastTripGold` on the same basis (purse + stash when no healing potion is carried), or add the
stash only when the town can withdraw. A `CanWithdraw()` probe on the town adapter is enough. Test: two consecutive
field ticks after a trip that bought nothing, stash 1418, expect `Futile`.

### F3 (medium) Mana reserve applies when the kit is already complete

`internal/town/shop.go:285` (V, from 20cd22ae). The reserve breaks out of the mana loop whenever `gold - cost < 130`
(`kitPotionGold + kitTPGold`). It does not look at whether a portal source and healing potions are already carried. By
then the shop has already bought the kit potions and the scroll (`shopVisit`: kit pass, then `buyFirstTP`).

Scenario: tome with 10 charges, 4 healing potions in the belt, 180 gold. Mana Potion at base 30 leaves 150, so the
first one is bought; the second is refused at 120. Before, the Sorceress bought as many mana potions as the gold
allowed. A mana-bound caster goes out with 1 or 2 mana potions. Two consequences follow. She rests more (regen pauses).
The trip trigger "mana-starved caster out of mana potions" (`internal/resource/town.go:57`) also fires sooner. Both
fit the drop in true XP/h after the 2nd run (H, U: needs the minute lines' `mana` and `potionsUsed` fields per run).

Smallest fix: reserve only the missing part of the kit:
`reserve = (no portal source ? kitTPGold : 0) + (healing carried < kitMinHP ? kitPotionGold : 0)`.
Test: kit complete, 180 gold, expect 2 or more mana potions bought.

### F4 (medium) `walkChainToTown` applies to every act 1 level, not to the listed fields

`internal/town/trip.go:174` (V). The guard is `Area >= data.Town(1)` (40), so the chain walk is tried from every act 1
area: Tamoe, the Monastery, Jail, Catacombs 1 to 4, Andariel's Lair (37), the Pit and Hole levels. `ToTown` is also
called outside the potion policy: the claim trip (`goal.go:85`), the heal trip (`goal.go:1416`), "Area done, returning
to town" (`goal.go:944`) and `rest`.

Scenario: Catacombs 4, no scroll, quest claim pending. Before, `ErrNoPortal` came back at once and the claim backed
off. Now the bot walks Catacombs 4 -> 3 -> 2 (waypoint only if active) through uncleared levels for up to 180 s,
then backs off (`claimBackoff`) and tries again. Worst case: 3 virtual minutes per claim attempt, with no potion.

Smallest fix: in `walkChainToTown`, return `noPortal` unless `resource.NearTownArea(area)` (or the new `chainHome`
of F1). Test: ToTown from area 37 with no scroll and no waypoint returns `ErrNoPortal` and never calls `ToArea`.

### F5 (medium) Recovery caps the visit that the withdrawal pays for

V. `advanceTrip` computes `PotionFloor` from the purse before the visit (`goal.go:428`, `broke.go:118`: pack gold
under `recoveryKitGold` 500 gives floor 2). The withdrawal then raises the purse to the keep line, 300 + 50 x clvl.

Scenario: the Gate 1 0338 case, clvl 11, purse 6, stash 1418. The bot withdraws 844, then buys at most 2 healing
potions and no mana (`RecoveryManaFloor` cap), plus a scroll. About 750 gold stays in the pack, where a death takes it
all (stashGoldBase note in docs/knobs.md). The next trip comes after 1 drink (2 healing < the threshold of 2 is false,
1 < 2 is true).

Smallest fix: compute `PotionFloor` after the withdraw step, or let the town ignore `PotionFloor` when the purse after
the withdrawal is at least `recoveryKitGold`. Alternatively withdraw only `kitGold()` (one scroll + `hpFirst` potions,
190 gold) while recovering.

### F6 (low) Withdrawal size and trigger

`internal/town/basicsfix.go:149,159` (V):

- `needWithdraw` fires when the purse is under the keep line, even if the purse already pays for the kit (purse 600,
  keep 850). That is an extra stash round (`openStash` plus `CloseAll`, U: about 5 to 10 s) and moves 250 gold out of
  the bank.
- `healingCarried` counts the belt (HP+RV) and pack HP but not pack rejuvenation. The bot side (`resInput`, `kitRead`)
  counts pack RV. A character with only a rejuvenation in the pack withdraws but is "not empty" for the policy.
  Harmless today (it only withdraws more), but the two counts should be one helper.
- No ping-pong: `bank` needs purse > keep, `withdraw` needs purse < keep, and the end-of-visit `stash` step deposits
  only above the same keep line (V). Stash full does not affect a withdrawal.

Smallest fix: withdraw only when `purse < kitGold()`. Amount `kitGold() - purse` while recovering, otherwise the keep
line. Share one healing count.

### F7 (low) The in-town 5 s gap: bounded, but the guard's cost is the round, not the gap

`internal/bot/goal.go:55-60`, `:313` (V). The emergency in-town gap is 5 s. After a visit that bought nothing, the
next tick is under 5 s (the stage-2 settle is `portalSettleMs` 600 ms), so the loop does not spin in town: it falls
through to travel and leaves. The 5 s gap therefore adds at most one extra chore round per town arrival (shop, stash
and withdraw walks, U about 20 to 40 s) when the shop cannot sell. It is not a loop by itself. The real
leave-and-return loop is F2's field side (60 s emergency gap, no futility).

The in-town test (`choresDue` with `kitGold >= minPotionPrice` = 30) uses `minPotionPrice`, a constant. `pickPotion`
then uses the base cost, and `buyUnit` refuses below 2 x cost (`shop.go:180`). With 30 to 59 gold and an `hp1` of
base 30, the shop can come out `Broke` every visit (U: the d2d vendor price of `hp1` is not checked here).

Smallest fix: `noteFutile`-style latch in town. After a visit with `rep.Broke` and healing still 0, no in-town
emergency until gold grows by `futileStep`.

### F8 (low) Cut-cast memory is lost when the context is already dead

`cmd/d2bot/d2dwire.go:291-310` (V). `afterCut` clears `cutAt` after its poll loop even when the loop never ran
(`ctx.Err() != nil` at entry). The next call within 8 s then casts again and spends the charge that the memory was
meant to save. The portal it finds is any town portal object within 25 units. Coming back to the field through the
old portal leaves that portal standing next to the player, so after a cut the 3 s wait can report "the cut cast opened
its portal" for the old one. `ToTown` then enters the old portal, which also leads to town, so this is harmless. Cost:
at most 3 s once per cut cast (V: `portalCutWait`, polled at `portalIdlePoll`). It is not paid on every cast.

Smallest fix: return early without clearing when `ctx.Err() != nil` at entry.

### F9 (low) BackToField retry doubles the worst-case portal wait

`internal/town/trip.go:308` (V). Two attempts, each with `enterPortal` plus `waitArea` 15 s, before the waypoint. The
worst case is about 30 s more than one attempt, then the waypoint or walk, inside `travelTimeoutSec` (300 s). This is
acceptable. The loop is bounded and checks `ctx.Err()`.

### F10 (low) Emergency now interrupts boss runs for kit trips

`internal/bot/goal.go:348` (ce5e7f1e) and `tripDue` (`goal.go:1350`) (V). A kit trip with 0 healing is an emergency,
and `tripDue` lets emergencies cancel a boss run, so Blood Raven or Andariel fights can be cut off for the walk home.
This is probably wanted when no potion is left. With no portal source, though, the walk from area 17 (Burial Grounds)
starts mid-fight.

Smallest fix: none needed. If unwanted, require a portal source for an emergency during `bossRun`.

## Items checked, no defect found

- HIREFIX. A dead merc with no unit gives `Extras.Merc{Dead: true}`, and `FromResult` reports Owned, not Alive; class
  is 0 because `merc.class` is not reported (V). GU/IW/0A map to `MercClasses[1..3]`, U live as the commit says.
  `ParseQuote` takes the digits after the last colon and skips `,`, `.` and spaces. "Resurrect Kyle: 1.234" reads 1234.
  A line with a colon in the name and no price reads the trailing name digits only when they exist. The affordability
  check uses pack gold, which is correct for the NPC.
- Context cancellation: `withdraw`, `walkChainToTown` and the `BackToField` loop all return on a dead `ctx` (V). No
  new goroutines, and the cut-cast state is behind `cutMu` (V). The guard calls `choresDue` from its own goroutine, and
  the new code there reads only the snapshot and `recoveryActive()`, both of which were read there before (V).
- Spending before healing: the shop order is still kit HP, then the scroll when `tpAllowed`, then HP, MP, RV (V
  `shop.go` `shopVisit`, `buyPotions`). The pre-existing exceptions are `ammo`, `beltbuy` and `repair`, which run
  before `shop` in `Chores` (`chores.go:298-303`) and are not part of these changes. Rejuvenation is not under the
  mana reserve.
- Way home: no change removes a way home. F1 and F4 add walks that are too long, not too few.

## Tests

- `TestTownArrivalRestocksWithoutCooldown`, `TestReplayGate1BeltWindowDecisions` and
  `TestRecoveryTownArrivalRestocksHealing` test the happy path at the decision level (`choresDue`). They would fail if
  reverted (V by reading: before beltfix3 the in-town visit was no emergency, and before beltfix4 `broke` switched the
  chore off). None of them covers the visit that cannot buy (F7), a repeated round, or the time cost.
- `TestBankedGoldIgnoredWithPotionOrNothingBanked` only tests the empty bank. The "with potion" half promised by its
  name is missing: a carried potion with stash 1418 must not count the bank.
- No test covers the side effects of the `walkableHome` edit on `AbortFight`, `PreEngage` or `TownTripTrigger` (F1),
  `walkChainToTown` from a non-listed area (F4), futility with stash gold (F2), or the mana reserve with a complete kit
  (F3).
- `TestPortalCutAfterClickReportsSpent` covers the reuse. It does not cover a dead context on the second call (F8) or
  a far portal.
- Build flexibility: the BELTFIX5 and BELTFIX3 tests use the default Sorceress only. AGENTS.md asks for a second build
  for economy rules. F3 in particular depends on the mana need (`resource.ManaNeed`).

## What to measure in the Gate 1 minute lines (ready-to-run, needs the run files)

1. Trips per hour by reason prefix (`town trip` reason) for the five runs, split by `NearTown` areas 4, 5, 6, 9, 10, 17
   and the rest (F1).
2. `walking the level chain to town` log lines: count, duration to arrival, and areas (F1, F4).
3. Mana potions bought per visit and `mana` at the minute line for runs 1-2 against 3-5 (F3).
4. Consecutive `town trip` lines with `emergency=true` and an unchanged `gold+stashGold` (F2).

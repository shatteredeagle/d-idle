# Review sweep 2, 2026-11-28: the OPEN rows of sweep 1 after the fixers merged

> status: current on 2026-11-28 (card sweep2, TASK-1128). A snapshot against main at `f58d585f2` (fix1b, vault2, live1, fix7b, nf1b, led1 merged). Continues `docs/probes/review-sweep-2026-11-28.md` (card sweep1), which is not edited here: its OPEN table is the input, this page is the status of each row now.

Source reviews: `docs/design/fable-review-2026-11-28/01` to `13`. Tags: V = re-read in the code or run as a test in this branch; S = taken from a card result or review text; U = not re-read in this run (the sweep 1 status stands).

Nothing here was run live. Every fix is offline (fakes), two classes where a class can matter.

## Fixed in this branch (agent/sweep2), one commit each

| finding | sev | what changed | must-fail test |
|---|---|---|---|
| 13 L2 (rest) | high | A boss chunk the board parked takes its boss area out of every leveling, farm and recovery pick (`Runner.parkedBossAreas`, called from `guardedAreasNoFarm`). The area comes from the margin query (`questboard.Encounter.Area`, filled from the danger boss table in `cmd/d2bot/survivalwire.go`). Run machine only; Try again or the pause policy gives the area back. The plan file is read at most every 10 s (U: judgment). | `internal/bot` `TestParkedBossChunkAreaIsNoLevelingCandidate` (Sorceress, Paladin) |
| 10 N4 (rest) | medium | `saveBookLocked` and `putEq` return the write error; a direct Equip or Wear set whose order cannot be written down is refused with code `journal` before the engine call (as Carry does). An open order that carries an unread reply is settled by the bytes in the slot at the next repair, like any other open order; the reply text stays in the order file. Before, such an order was open for ever. | `internal/vaultkeep` `TestSweep2DirectEquipRefusedWhenTheJournalCannotBeWritten`, `TestSweep2OrderWithUnreadReplyIsSettledByTheLook` |
| 10 N6 (first half) | low | "This order filed it" matches the order tag on a record in any state, before `Add`. | `internal/vaultkeep` `TestSweep2DisplacedItemIsFiledOncePerOrderInAnyState` |
| 02 #8 (first point) | medium | Hand-in reads the evidence after every attempt, also one that returned an error: a dialogue that paid and then failed is done after one attempt. | `internal/questboard` `TestSweep2HandInThatPaidAndErroredIsDone` |
| 10 B3 (first half) | low | The hand-in park seeds the run state from the stored counters (`Exec.seeded`, shared with `Report`). | `internal/questboard` `TestSweep2HandInParkKeepsStoredDeaths` |
| 11 N8 | low | `earlyRouteCheck` and `admit.Rows.MayEnter` ask every road row's level and life before any row's kit. The differ cell "words" has its counter (`AdmissionExplain.Words`). | `internal/bot` `TestEarlyRoadAsksEveryFloorBeforeAnyKit`, `TestAdmissionShadowCountsTheWordsCell` |
| 07 admission (new, found here) | medium | The early road kit became an entry rule in the old judge only (nf3, TASK-1104): the function refused a hero already on the road, a new-only cell at every ask, and `bot.admission` 1 would have parked the Maggot Lair chunk again. `admit` now skips the kit of a road the hero stands on. | `internal/bot` `TestAdmissionKitIsAnEntryRuleLikeTheOldJudge` (both modes, two classes) |
| 11 N1c (decline cap) | medium | The bot reports a hook that had nothing to do with `Shadow.Decline`; its back-off is capped at `arbiter.DeclineMax` (120 s, one lease; U: judgment, a constant, no knob). A failure after cost keeps 1800 s. | `internal/arbiter` `TestDeclineBackoffIsCapped`; `internal/bot` `TestArbiterDeclinedHookIsAskedAgainAfterTheCap` |

Build-specific part: none. The only class-dependent data read is the road floors in `internal/plandata/admission.json` and `earlyroutes.json` (unchanged).

## Stale or closed by the fixers (V unless tagged)

| finding | status now |
|---|---|
| 10 N4 unreadable slot, empty slot, two-hander | fixed-by fix1b (bytes verdict, unreadable slot keeps the order open) and vault2 (other-hand pre-read); S card results, V the tests `TestFix1b*`, `TestVault2TwoHander*` pass |
| 10 N5 | fixed-by fix1b (S card result) |
| 10 B2 | fixed-by fix1b (S card result) |
| 13 L1, L2 (hole walk), L3 (board side), L4, L5, L6 | fixed-by live1 (S card result; V `holeHoldWhy`, `samePathWindow` in the code) |
| 11 N1 (a, b), N2 to N6 | fixed-by fix7b (S card result) |
| 03 #5 "escape and whirl landings" | already gated (sweep 1 correction stands) |

## Still open, with the reason it was not fixed here

| finding | sev | status | reason |
|---|---|---|---|
| 10 N8 / 01 F6 twin Keep | medium | OPEN (V `keep.go` `twinDoubt` unchanged, still no test that names it) | changes what "kept count" means (mark the displaced twin's record returned): a vault state change that wants its own card and a fork-side look at which copy is on the hero |
| 10 N6 (second half) | low | OPEN | `createBack` keys are per game process: needs "placed" stored in the order, and whether the fork keeps keys over a restart is needs-fork (`docs/design/fork-order-id-spec.md`) |
| 10 N4 owner action "I checked, close it" | low | needs-owner | a new desk action; the look now settles the order, so the action is only for the case where the look can never be taken |
| 10 B1 deaths over the life of a run | medium | needs-owner | window or documented behaviour |
| 10 B3 (second half) | low | OPEN (U) | counters when `smPark` cannot save: not reached for time |
| 10 B4 go taken before the hop is verified | low | needs-owner | the review calls it an opinion; a refused difficulty then keeps the go and the ban rule changes |
| 10 section 4 rung 4 pairing test | medium | OPEN (U) | not reached |
| 03 #5 (rest) escape "and" against "or", hop cooldown | medium | needs-owner (SME rule) | unchanged |
| 02 #8 (rest) | medium | OPEN (U) | a tapper that leaves mid-dialogue; hand-in under gold farm, recovery and the arbiter hold is a doc line |
| 02 #9, 02 smaller | low | OPEN (U) | not reached |
| 11 N7 stance | medium | needs-owner (SME per-kind floor); dark (V no row carries `enabled`) | unchanged |
| 11 section 4 / 07 A4 slice budget | medium | OPEN, needs a data row | fix7b documented it |
| 07 A5, A6 | medium | design (TASK-973) | unchanged |
| 07 A7 aura dwell | medium | OPEN (U) | not reached |
| 07 A8 | medium | needs-live | one control-0 recorded night |
| 07 A9 | low | OPEN (U) | not reached |
| 07 admission (rest) | medium | OPEN | the exhaustive old-against-new equality test, `fillEarly` reading the function, `View.Knobs` and `View.Legacy` unfilled (V `admitView`), unused purposes. The on-road difference found here says the exhaustive test is worth its own card before `bot.admission` 1 |
| 13 L3 nav start cell | high | needs-live | the recorded Tower Cellar 3 grid (`docs/probes/cellar3-gate-capture.md`); live1 tried a second pocket cell on the synthetic maps and reverted it |
| 13 L4 vendor pays an owner-only reward | medium | needs-owner | live1 criterion 6 |
| 13 L9 | low | OPEN (U) | not reached |
| 12 C2 flip run check | medium | OPEN (V `tools/flip/flip.py` still reads `printenv` and `/proc/1/environ` only) | Python tool with an ops half (watcher revert, autodeploy); not reached |
| 12 C3 (rest), C4, D2, D3, E3, low 5, 8, 9, 10 | medium, low | OPEN (V for low 8: `cmd/d2scenario/golden.go` still drops the `scenarioHero` error; the rest U) | not reached |
| 01 F7, F8, F10, F11 | medium, low | OPEN (U); F8 ids needs-fork | not reached |
| 04 #6, #9, #10, #11 | medium, low | OPEN (U) | not reached |
| 06 E3, E7, E8, fix 9, 10; 06 F1 to F5; 05 1.2 to 1.6 | high to medium | OPEN rules; needs-owner for the flake rule and the matrix floor; `tools/flip` and ops | unchanged |
| 08 fix 2 | medium | needs-live | unchanged |
| 08 fix 7 | low | OPEN (U) | unchanged |

## Interface notes

- `questboard.Encounter.Area` (new, additive): the area of a boss encounter, 0 when not known. A margin query that does not fill it switches the parked-boss exclusion off for that boss; nothing else reads it.
- `arbiter.Arbiter.Decline` and `Shadow.Decline` (new): "the hook had nothing to do". `Report(in, now, false)` keeps meaning a failure after cost.
- `vaultkeep`: a direct Equip can now answer code `journal` (the page already shows that code for Carry and Stow).
- `questboard.Fake.HandInLateErr` (test fake): a hand-in that registers and still returns an error.

## Live checks (not run)

1. Sandbox hero with a parked boss chunk (`ops quest status` shows `blocked`): no `zone choice` line names the boss area until Try again.
2. `bot.arbiterControl` 1 canary: after declined refills the `arbiter` line shows `backed off` of at most 120 s.
3. Direct Equip with the state directory read-only: the page answers "could not be written down first", the hero wears what it wore.

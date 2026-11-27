# townflip: finishing and flipping town.plan (roadmap P5, TASK-972)

> status: current (2026-11-26). Runbook and state of the knob `town.plan` stage by stage. Design: [town-planner.md](fable-review-2026-11-21/town-planner.md); rule: [dark-launch.md](../process/dark-launch.md); ADR 0028 (owner looks before the purse, sell and route stages).

Tags: V verified (how is said), S read in code or ledger, U unverified.

## 1. Where each stage stands

| Stage | What it does | Code | Evidence | Flip state |
|---|---|---|---|---|
| 1 shadow | logs `town plan (shadow)`, changes nothing | merged (town3) | S | superseded by 2 |
| 2 guards and shop | guards, chunk preconditions and the shop read the plan (belt plus pack) | merged (town4) | S ledger row `town.plan`: trial value 2 on the dev sandbox, 45 min window in bundle 2 (accelerated soak), closed `enabled` | ON in one sandbox (the dev sandbox hero). The second (farm) sandbox and a 2 h window there are NOT recorded: card criterion 1 open |
| 3 purse | the purse is settled once per visit to the plan's spend plus the carry line; recovery and poverty read wealth | merged (town5; card still In Progress) | S town5 result: unit and rig tests pass, scenario broke-after-bank green at stage 2 code, malus-errand-gold blocked twice by the save point | dark; owner look needed (ADR 0028) |
| 4 sell and pickup | sell keeps what the plan line needs; pickup wants while under the same keep | built: sell side (townflip, `internal/town/sellplan.go`), pickup side (town6: `loot.SetPotionPlan`, `town.pickupKeep`) | V offline: `go test ./internal/town -run 'PlanSellKeep\|PlanKeep\|PickupKeep\|Stage4'` on two builds (Sorceress, Paladin), the rig tests with the game tables on the worker host; U live: scenarios belt-refill-from-pack, sell-keeps-need, scroll-hoard not run | dark; ready for the flip sequence once the three scenarios are green and the owner has looked |
| 5 route | stops ordered by nav cost | built (town8, TASK-761, merged 2026-11-27): `town.plan >= 5` orders the enabled chores by the cheapest walk under precedence rules (`internal/town/chores.go`); short-gold priority table as data (town7, TASK-759: `internal/supply/priority.json`) | S code read (`PlanStage() >= 5` in chores.go) and card result; offline only, U live (no stage 5 scenario run) | dark; ready for the flip sequence once one-visit and determinism scenarios are green and the owner has looked |
| cleanup | delete stage 0, old counters and retired `resource.*` knobs | card town11 (TASK-764, Proposed: blocked until stages 3 to 5 are default) | none | after stage 5 holds one release |
| view and arbiter | `GET /api/townplan` and the Town plan card (town9, TASK-762, merged 2026-11-27); upkeep source for the arbiter (town10, TASK-763) | S card results, offline only | none | read-only, not a stage |

## 2. Stage 4 sell side (this branch)

At `town.plan` 4 or more the sell chore (`sellableIf`) first solves the plan of the visit (the chores call's supply profile, else `field`) and protects, per line, `max(target, min)` potions counted belt plus pack: the belt is never sold, so the pack keeps the line's remainder after the belt (healing also subtracts every rejuvenation), best tier first, ties by unit id. Every other healing or mana potion is surplus under pack pressure (replaces `town.spareKeep` for potions only; scrolls, keys and tomes keep their old allowance). A player's `sell` mark still wins; a `keep` mark too. Rejuvenations stay with `loot.rejuvSpare`. A plan that cannot be solved logs `town plan (sell): ...` and falls back to today's rules. Build-specific part: none in Go; the lines come from `internal/supply/profiles.json` and the belt plan of the class.

## 2b. Stage 4 pickup side (town6)

At `town.plan` 4 or more the town installs `pickupKeep` on the Looter it shares (`town.New`, `SetJudge`): a ground healing or mana potion that the belt cannot take is wanted while the pack holds fewer usable potions of its kind (tier at or above the floor of `internal/loot/potionfloor.json`) than the plan line's keep, the same `planKeepOf` number the sell chore protects, so a potion picked is one the sale keeps. The plan is solved for the supply profile of the last chores visit (the excursion the hero was filled for; `field` before any visit). A kind without a plan line (mana in profiles without a mana line, or for builds whose mana need is under the row's `manaNeedAtLeast`) keeps today's buffer `loot.potionBufferMana`; then the sell chore may still sell those under pack pressure (known gap, harmless). With a full belt the healing keep is the pack reserve of the kit clamp (2), not `loot.potionBuffer` (4): the pickup stops earlier than at stage 0, by design of the plan. The belt-landing rule and rejuvenations are unchanged; scrolls and keys keep their current caps (the plan's portal line target is only its min, so reading it would refuse useful scrolls; S read in `internal/supply/profiles.json`). A plan error logs `town plan (pickup): ...` once per distinct error and the buffers decide. Build-specific part: none in Go.

## 3. Flip sequence (one stage at a time, coordinator)

For each stage N (2 on the farm sandbox, then 3, then 4 (town6 merged), then 5 (town8 merged)):
1. Precondition: the stage's scenarios green (town5 malus-errand-gold for 3; town6 trio for 4; town8 one-visit and determinism for 5), the owner has looked (stages 3 to 5), no open window on the sandbox.
2. Dry run, then flip: `python3 tools/flip/flip.py <sandbox> town.plan=N --why "<scenario> green" --window 2h --dry-run`, then without `--dry-run`.
3. Watch: `tools/agentrun.sh <agent> --ttl 90000 -- python3 tools/flip/flip.py watch`; a hard breach reverts on its own and the row goes back to `dark` with `revert_reason`.
4. Counts: record the invariant counts before and after (section 4) in the card result.
5. After a clean window on both sandboxes, the coordinator flips the code default (`internal/env/tuning_town.go`, `Plan`) to N and sets the row `default-on`.

## 4. Live check for a stage on the dev sandbox (ready to run, not run by this card)

- `d2bot ops status --why` shows `town.plan` N under `knobs`; the minute line carries `townPlan N`.
- Stage 2: chores log lines `town plan (shadow): ... verdict=` and no guard refusal naming a min line the plan can close.
- Stage 3: one `town plan: purse keep=K spend=S ...` line per visit; the purse after the stash stop is within one potion price of K plus S.
- Stage 4: after a sell visit the pack still holds the plan's healing and mana remainder (best tier); no `town plan (sell):` or `town plan (pickup):` warning; `loot: potion skip ... buffer=N` lines show the plan keep N (1 or 2 with a full belt), not 4.
- Hard rows of the invariant monitor (potion-not-drunk, life-zero-alive, refusal-storm, xp-frozen, mode-no-exit, corpse-unresolved) stay at their baseline for the 2 h window (user decision 2026-11-27: no 24 h waits).

| Stage | Sandbox | Window | Hard breaches | Result |
|---|---|---|---|---|
| 2 | dev | 45 min (bundle 2) | 0 (S ledger `enabled`) | a 2 h window on the second sandbox is still open |
| 2 | farm | not run | | |
| 3, 4, 5 | | not run | | |

## 4b. Stage 4 scenarios: ready-to-run checklist (not run; needs a lab slot, CHEATED setups, mechanics only)

Definitions: `tools/provingground/scenarios/townplan.json` (ids `town.plan.*`, Paladin and Sorceress, seeds 11 and 3, `no_writes`); offline numbers for each are in `internal/town/sellplan_test.go` and `pickupplan_test.go` (V, `go test -p 1 ./internal/town`). Run each with `town.plan=4` set by the scenario, once with `town.plan=0` as the control.

| Scenario | Tier | Green means | Control (plan 0) |
|---|---|---|---|
| `town.plan.belt-refill-from-pack` | T2, T3 | belt full for healing and mana after the visit; `item_in_pack hp2` at least capacity plus buffer; one `town plan: ... sell=` line | U, expected RED on the keep count |
| `town.plan.sell-keeps-need` | T3 | after the first visit `item_in_pack hp2` at least the plan target; no `tower guard (healing` line | RED (the sell chore dumps the need) |
| `town.plan.scroll-hoard` | T5 | `item_in_pack tsc` at most 2, tome kept, no `town portal did not appear`; the ground half (full tome refuses scrolls) is U | green in town, ground half U |

A scenario that cannot reach its setup state is a scenario fault, not a finding. The stage 4 flip waits for these three green (section 3).

## 5. Dead ends and notes

- The card owns the flip, not the stage code; stage 4's pickup side and stage 5 were built by town6 and town8 so two agents do not edit `internal/loot` and the route at once.
- Before town6 the sell keep and the pickup buffer could disagree; both now read `planKeepOf`. Mana in a profile without a mana line is the one remaining split (pickup by buffer, sell by plan), see section 2b.

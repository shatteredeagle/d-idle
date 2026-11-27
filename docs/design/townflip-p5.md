# townflip: finishing and flipping town.plan (roadmap P5, TASK-972)

> status: current (2026-11-26). Runbook and state of the knob `town.plan` stage by stage. Design: [town-planner.md](fable-review-2026-11-21/town-planner.md); rule: [dark-launch.md](../process/dark-launch.md); ADR 0028 (owner looks before the purse, sell and route stages).

Tags: V verified (how is said), S read in code or ledger, U unverified.

## 1. Where each stage stands

| Stage | What it does | Code | Evidence | Flip state |
|---|---|---|---|---|
| 1 shadow | logs `town plan (shadow)`, changes nothing | merged (town3) | S | superseded by 2 |
| 2 guards and shop | guards, chunk preconditions and the shop read the plan (belt plus pack) | merged (town4) | S ledger row `town.plan`: trial value 2 on the dev sandbox, 45 min window in bundle 2 (accelerated soak), closed `enabled` | ON in one sandbox (the dev sandbox hero). The second (farm) sandbox and a full 24 h window are NOT recorded: card criterion 1 open |
| 3 purse | the purse is settled once per visit to the plan's spend plus the carry line; recovery and poverty read wealth | merged (town5; card still In Progress) | S town5 result: unit and rig tests pass, scenario broke-after-bank green at stage 2 code, malus-errand-gold blocked twice by the save point | dark; owner look needed (ADR 0028) |
| 4 sell and pickup | sell keeps what the plan line needs; pickup wants while under target | SELL SIDE built here (`internal/town/sellplan.go`); pickup side not built (card town6) | V offline: `go test ./internal/town -run 'PlanSellKeep\|PlanKeep'` on two builds (Sorceress, Paladin); U live | dark; do not flip before town6 adds the pickup side and its scenarios (belt-refill-from-pack, sell-keeps-need, scroll-hoard) |
| 5 route | stops ordered by nav cost | not built (card town8) | none | dark |
| cleanup | delete stage 0, old counters and retired `resource.*` knobs | card town11 | none | after stage 5 holds one release |

## 2. Stage 4 sell side (this branch)

At `town.plan` 4 or more the sell chore (`sellableIf`) first solves the plan of the visit (the chores call's supply profile, else `field`) and protects, per line, `max(target, min)` potions counted belt plus pack: the belt is never sold, so the pack keeps the line's remainder after the belt (healing also subtracts every rejuvenation), best tier first, ties by unit id. Every other healing or mana potion is surplus under pack pressure (replaces `town.spareKeep` for potions only; scrolls, keys and tomes keep their old allowance). A player's `sell` mark still wins; a `keep` mark too. Rejuvenations stay with `loot.rejuvSpare`. A plan that cannot be solved logs `town plan (sell): ...` and falls back to today's rules. Build-specific part: none in Go; the lines come from `internal/supply/profiles.json` and the belt plan of the class.

## 3. Flip sequence (one stage at a time, coordinator)

For each stage N (2 on the farm sandbox, then 3, then 4 once town6 lands, then 5 once town8 lands):
1. Precondition: the stage's scenarios green (town5 malus-errand-gold for 3; town6 trio for 4; town8 one-visit and determinism for 5), the owner has looked (stages 3 to 5), no open window on the sandbox.
2. Dry run, then flip: `python3 tools/flip/flip.py <sandbox> town.plan=N --why "<scenario> green" --window 24h --dry-run`, then without `--dry-run`.
3. Watch: `tools/agentrun.sh <agent> --ttl 90000 -- python3 tools/flip/flip.py watch`; a hard breach reverts on its own and the row goes back to `dark` with `revert_reason`.
4. Counts: record the invariant counts before and after (section 4) in the card result.
5. After a clean window on both sandboxes, the coordinator flips the code default (`internal/env/tuning_town.go`, `Plan`) to N and sets the row `default-on`.

## 4. Live check for a stage on the dev sandbox (ready to run, not run by this card)

- `d2bot ops status --why` shows `town.plan` N under `knobs`; the minute line carries `townPlan N`.
- Stage 2: chores log lines `town plan (shadow): ... verdict=` and no guard refusal naming a min line the plan can close.
- Stage 3: one `town plan: purse keep=K spend=S ...` line per visit; the purse after the stash stop is within one potion price of K plus S.
- Stage 4: after a sell visit the pack still holds the plan's healing and mana remainder (best tier); no `town plan (sell):` warning.
- Hard rows of the invariant monitor (potion-not-drunk, life-zero-alive, refusal-storm, xp-frozen, mode-no-exit, corpse-unresolved) stay at their baseline for 24 h.

| Stage | Sandbox | Window | Hard breaches | Result |
|---|---|---|---|---|
| 2 | dev | 45 min (bundle 2) | 0 (S ledger `enabled`) | needs the 24 h window |
| 2 | farm | not run | | |
| 3, 4, 5 | | not run | | |

## 5. Dead ends and notes

- The card owns the flip, not the stage code; stage 4's pickup side and stage 5 stay with town6 and town8 so two agents do not edit `internal/loot` and the route at once.
- A stage 4 flip with only the sell side built would make the sell keep and the pickup buffer disagree (the pickup may still refuse a potion the sell keep would hold); that is safe (fewer potions picked, none sold below target) but it is not the finished stage.

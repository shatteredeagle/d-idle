# Input: cast verbs (cast-unit, cast-point, cast-hold)

> status: bot side built offline (TASK-726, card dlv-cast-verbs, 2026-11-27), dark behind `input.planner` (0) and the capabilities `input.cast-unit`, `input.cast-point`, `input.cast-hold`, which no fork build lists yet. Nothing here ran against a game. Tags: V verified (how is said), S secondary, U unverified.

Design: [combat-engine.md](fable-review-2026-11-21/combat-engine.md) section on input (table of cast messages, question Q3), card row in [appendix D](fable-review-2026-11-21/combat-engine-d-cards.md) wave 2. Rule of [ADR 0021](../adr/0021-input-rule-verb-key-click.md): verb, then key, then verified click; a verb is promoted to input only after its differential. Same capability pattern as [input-i3.md](input-i3.md).

## 1. What is built

| piece | file |
|---|---|
| capability names, `input.Cast` (skill, hand, aim, unit, hold), `Caster` and `HoldCaster`, `Planner.RunCast` | `internal/input/castverbs.go` |
| `Action.Once` and an optional `Verify` in the planner | `internal/input/planner.go` |
| `d2dinput.Controller.CastUnit`, `CastPoint` (`cmd skill <id> x y [unit] [left]`) | `internal/d2dinput/castverbs.go` |
| `d2dcombat.Input.Deliver(id, use, aim, hold)`: hand, target and press from the capability row's `use` object; a corpse target carries the nearest corpse's unit id (no corpse near the aim: nothing is sent); `CastAt` and `Swing` go through the same `send` | `internal/d2dcombat/castverbs.go`, `input.go` |
| candidate caps file of the fork branch | `data/input/caps/d2d-fork-cast-verbs.json` |
| scenario `input.cast-point-lands` (pending) | `tools/provingground/scenarios/input-scenarios.json` |

Rules (V by the tests in the three packages): the verb goes out only with `input.planner` on AND the build listing the request's capability; with the knob off the old command goes out and nothing is counted, so the minute line `input=v/k/c` is unchanged. A verb that was NOT sent (error reply, older build, no verb on the backend) falls back to the old command once. A verb that was sent is the cast: it is never followed by a second cast by another path, whatever its effect; the expectation ledger of dlv-expect judges the effect later. On d2d the old command and the verb are the same `cmd skill` line (the fused verb has always been the d2d cast path, V code read), so what the capability changes is that the cast is counted as a verb and that the left hand and the corpse unit are reachable through one request.

Nothing is build-specific: hand, target kind and press are read from `data/capabilities/capabilities.json`. Tests pick rows by class and target, naming no skill: Sorceress and Paladin for unit and point casts, Paladin and Barbarian for the hold fallback (the Sorceress has no repeat or channel row). No class or skill literal is added to Go.

Not done here (follow-up, outside this card's `owns`): the Fighter still calls `In.CastAt`, `In.Click` and `Swing` in `internal/combat/fighter.go` `cast`; moving it onto `Deliver` with the expectation as `Verify` belongs to the card that owns that file (dlv-aim-point is the next one in wave 2).

## 2. Hold

No fork build has a hold verb (V by reading: no caps file lists `input.cast-hold`, and `docs/d2d-combat.md` item 5 records hold skills as unsupported on d2d). `d2dinput.Controller` therefore does not implement `input.HoldCaster`, and a repeat or channel press falls back to one `cmd skill`, which is what d2d did before. Fork change proposed (described here, not applied; patches never go into this repo; branch `didle/cast-verbs`): a verb `cmd skillhold <id> <x> <y> <unit> <frames> [left]` that sets the hand's skill, sends the same use-skill message the held button sends on every attack-sequence restart for `frames` game frames, and ALWAYS releases at the end, also when the hero dies, changes level or a refusal ends it; reply `ok held=<frames sent> casts=<n>` or `err <refusal>`; listed in `caps` as `input.cast-hold` only after a differential against a held mouse button (same casts, same frames). The bot side then only needs `CastHold` on the controller.

## 3. Differentials (to run in the fork; not run here)

Pattern of `tests/input_items_diff_d2d.py` (two headless d2d, same seed and setup, the verb on one, the other path on the other, digest compared after each step). Checklist for the fork owner:

1. Q3, fused against select then cast: run A `cmd skill <id> x y <unit>`; run B `cmd select <id> 0`, one tick, then the right click on the unit. Unit target (a Sorceress bolt), point target (Blizzard), left hand (a Paladin Smite with `left`). Digest: mana, the skill on each hand after the step, missiles (owner, skill, cell), monster life. Expect 0 differences apart from the one tick; if the hand differs after the step, the fused verb must restore the old hand or Q3 fails and the verb is split.
2. Point lands: `cmd skill <point id> x y` with no unit at a free cell 5 cells away; the missile or area object appears within one cell of x, y in both runs. Also a cell with a monster on it (no unit given): the cast goes to the cell, not to the unit.
3. Corpse: a corpse skill with the corpse's unit id against the click on the corpse; refusal when the corpse is used up moves nothing.
4. Refusals that must move nothing in both runs: not enough mana, a skill on cooldown, a dead hero.
5. When 1 to 4 pass: add `input.cast-unit` and `input.cast-point` to the fork's `caps` list and record the result in the note of `data/input/caps/d2d-fork-cast-verbs.json`.

## 4. Enable criteria

`input.planner` stays 0 until a build lists the cast capabilities after section 3 and the scenario `input.cast-point-lands` passes with writes disabled on Sorceress and Paladin. Live checklist (not run): hidden instance, throwaway character, `input.planner=1`, a fight in Blood Moor, the bot log shows `input path action=cast-point path=verb` and no `input path failed action=cast-point`, the hero left in town with no panels open.

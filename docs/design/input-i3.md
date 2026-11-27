# Input layer I3: town portal by verb; stat and skill points by verb

> status: bot side built offline (TASK-986, card inputl3, 2026-11-27), dark behind `input.planner` (0) and the backend capability `input.use-item`, which no fork build lists yet. Nothing here ran against a game. Tags: V verified (how is said), S secondary, U unverified.

Roadmap R17, row I3 of [07-input-layer](fable-review-2026-11-19/07-input-layer.md) section 6; rule of [ADR 0021](../adr/0021-input-rule-verb-key-click.md): verb, then key, then verified click; each verb promoted from lab to input needs its own differential against the game's message.

## 1. Town portal

**Which message.** A right-click on a Scroll or Tome of Town Portal in the pack sends the client's use-item message (C->S 0x20). The fork's `cmd use <item id>` sends exactly that message (`cmd::UseItem`), and the world hands `tsc` and `tbk` to `World::read_portal`, the same handler the click reaches (V: fork source read at the 2026-11-25 rebase, `apps/d2d/devctl/verbs.hpp` the `use` branch, `components/game/world/commands.cpp` the `UseItem` branch, `components/game/world/travel.cpp` `read_portal`). The 07 review's wording "`cmd skill` with the tome as the item-owned skill" does not fit d2d: `cmd skill 220 ... <item>` matches only charged skills (`commands.cpp`, the `use_skill.item` branch) and did nothing in the lab (V-d2d, docs/d2d-town.md). The world casts the item-owned skill itself after the use-item message, as Game.exe does, so the verb is `cmd use`.

`read_portal` refuses silently in town, in Uber Tristram, when dead, when the hero cannot be interrupted, with an item on the cursor and for an empty tome (V source read). A verb is therefore a request; the effect is read back.

**Bot side** (built here):

| piece | file |
|---|---|
| capability name `input.use-item`, interface `input.ItemUser`, `input.AsItemUser` (passes through the d2d town's hover wrapper) | `internal/input/useitem.go` |
| `d2dctl.Client.CmdUse` (`cmd use <id>`), fake accepts it | `internal/d2dctl/use_verb.go` |
| `d2dinput.Controller.UseItem` | `internal/d2dinput/useitem.go` |
| combat safety portal: verb first, verified by the portal object near the hero, the scroll gone, or the tome's quantity stat (70) lower than before | `internal/combat/portalverb.go`, called at the top of `portalAttempt` |
| town trip portal (`ToTown`): verb first, verified by the portal object near the hero; on failure the log line carries the tome quantity before and after | `internal/town/portalverb.go`, called in `openPortal` |

Rules, both sites: the verb is used only with `input.planner` on AND the connected build listing `input.use-item` AND a controller that has the verb. A send error (older build, transport) falls back to the pack click for that try. A verb that was sent but opened nothing is an error for that try and is NOT followed by the pack click (the click is the dropped-cast incident path); the existing retry loop sends the verb again. A verified verb counts as `verb` in the minute line `input=v/k/c`. Nothing is build-specific: the portal item is chosen by the item table (`Portal`, `Tome` flags), the same for every class; tests run Sorceress against Paladin, Necromancer and Barbarian.

## 2. Stat and skill points

Already on verbs on d2d before this card (V code read): `cmd stat` and `cmd skillpt` send `cmd::StatPoint` and `cmd::SkillPoint`, the messages the + button and the skill icon send; the leveler spends through them with read-back and falls back to clicks at the first failure (`internal/leveler/leveler.go` `runVerbs`, `cmd/d2bot/pointswire.go`, docs/points.md section 3). No bot change was needed. What is missing is the differential (section 3) and caps tokens (docs/process/points-verbs-devp14.md item 1).

## 3. Differentials (to run in the fork; not run here)

The pattern of `tests/input_items_diff_d2d.py` (two headless d2d, same seed and setup, the verb on one, the pointer path on the other, a digest compared after each step; a non-refusal step must change the verb run, a refusal must move nothing). That file already has a `use` step for a healing potion (19 steps, 0 differences, V per docs/process/sandbox-input-items.md) but no town portal and no points step. Ready-to-run checklist for the fork owner:

1. TP scroll, field (Blood Moor): `cmd use <tsc id>` against `rclick` on the scroll cell with the pack open. Digest adds `debug portals` (levels and count, not position) and items. Expect: scroll gone, one portal pair, 0 differences.
2. TP tome with 3 charges, field: the same against the tome cell. Expect: tome charges 3 -> 2 on both sides, one portal pair.
3. Second cast from the tome: the old pair closes, one new pair (both sides).
4. Refusals that must move nothing: in town; with an item on the cursor; an empty tome (charges 0).
5. Stat point: `cmd stat 0 1` against a click on the Strength + button with the character panel open (setup `debug statpts 5`). Digest: `stats.base`, `statpts`.
6. Skill point: `cmd skillpt <index>` against a click on the skill icon (setup `debug skillpts 2`, a level-1 skill). Digest: `skills.base_levels`, `skillpts`. Refusal: a skill whose required level is not met moves nothing.
7. When 1 to 4 pass: add `input.use-item` to the fork's `caps` list (`apps/d2d/devctl/verbs.hpp`, the `list` string) and to a `data/input/caps/<build>.json` here with the differential's result in its note.

## 4. Enable criteria

`input.planner` stays 0 (ledger row input.planner) until the fork lists `input.use-item` after steps 1 to 4 and a live drill passes with writes disabled: the hero opens a town portal from a scroll and from a tome in the field by verb (minute line `input=` with v > 0 and no inventory panel flash), on two classes. Live checklist (not run): hidden instance, throwaway character with a scroll and a tome, `input.planner=1`, cast in Blood Moor three times, read `debug portals` and the tome charges after each cast, leave the hero in town with no panels open.

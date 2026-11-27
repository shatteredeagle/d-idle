# Input layer I4: fork verbs buy, sell, repair, take-waypoint, menu-choose

> status: client side built offline (TASK-988, card inputl4, 2026-11-27), dark behind `input.planner` (0). The fork half is NOT done here (no fork edits from this card): section 3 lists what the fork owner runs. Live: U. Tags: V verified (how is said), S secondary (read from source or docs), U unverified.

Roadmap R17, row I4 of [07-input-layer](fable-review-2026-11-19/07-input-layer.md) section 6; rule of [ADR 0021](../adr/0021-input-rule-verb-key-click.md): verb, then key, then verified click; a verb is promoted from lab to input only after its own differential against the game's message.

## 1. State of each verb on the client side

| verb | fork command, message | caps name | client (d2dctl) | planner side | adopted by |
|---|---|---|---|---|---|
| buy | `buy <stock> [tab] [fill]`, 0x32 | `input.buy` | `VerbBuy` (V: tests, fake) | `input.WorldVerbs.BuyAt` | `internal/town/shop.go` (before this card) |
| sell | `sell <id>`, 0x33 | `input.sell` | `VerbSell` | `WorldVerbs.SellItem` | `internal/town/shop.go` (before) |
| take-waypoint | `waypoint <level>`, 0x49 | `input.waypoint` | `VerbWaypoint` | `WorldVerbs.TakeWaypoint` | `internal/nav/waypoint.go` (before) |
| repair | `cmd repair <item\|-1>`, 0x35 | `input.repair` | `CmdRepair` (new) | `input.Repairer`, `input.AsRepairer` (new, `internal/input/repairverbs.go`) | `town.pressRepairAll` in the repair chore (new) |
| menu-choose | `menu choose <text>`, client UI branch | `input.menu-choose` | `MenuChoose` (new) | `input.MenuChooser`, `input.AsMenuChooser` (new) | nothing yet (section 4) |

`d2dinput.Controller` implements all of them; `d2dtown.hoverFirst` passes them through its world-verbs holder, so the town sees them behind the wrapper.

Rules of the repair chore (V by `internal/town/repairverb_test.go`, Sorceress and Barbarian): the verb goes out only with `input.planner` on AND the build listing `input.repair`; otherwise the Repair All click as before. A verb error (older build) falls back to the click for that try. The durability read-back is unchanged and decides success: a verb that the world dropped silently shows as "repair not confirmed", the same latch as a dropped click. A sent verb counts as `verb` in the minute line `input=v/k/c`. Nothing is class-specific; the two classes in the test prove that.

Lines sent (V: `internal/d2dctl/use_verb_test.go`, fake d2dctl): `cmd repair -1`, `menu choose Trade`; text with a newline is refused before sending.

## 2. Candidate caps

The fork's caps set `data/input/caps/d2d-fork-forkcaps.json` already lists all five names (and `input.use-item`, casts). The bot adds no caps row of its own; the sandbox build that loads it decides.

## 3. Fork half: what the fork owner runs (no fork edit was made here)

On branch `didle/forkcaps-20261127` (private remote, same-day push rule). One differential each, pattern of `tests/input_items_diff_d2d.py` (two headless d2d in lockstep, the same seed and setup, the verb on one, the client's own path on the other, a digest after each step, a non-refusal step must change the verb run, a refusal must move nothing):

1. repair all: worn wand 7/15, `cmd repair -1` against a click on the Repair All button with the smith window open. Digest: worn durability, gold. Expect equal durability and equal gold spent.
2. repair one item: `cmd repair <id>` against the single Repair button then a click on the item. Refusals: not enough gold; nothing to repair; a non-repairable item (a belt item, a potion).
3. menu-choose: `menu choose Trade` against a click on the dialog line, for Akara and Charsi; digest the UI flags (vendor window). Refusal: no dialog open.
4. buy, sell, take-waypoint: their differentials are recorded in docs/process/input-verbs2.md (S, read there); re-run only if the branch changed them.
5. When 1 to 3 pass, update the note of `data/input/caps/d2d-fork-forkcaps.json` with the result (a card that owns that file).

Known gap (U): docs/process/sandbox-devp14.md records that the headless hero never reached the vendor, so `cmd repair` and `menu choose` were never run live; steps 1 to 3 must walk the hero with `debug goto` first.

## 4. Not done, on purpose

- menu-choose has no adopter. The town's `pickEntry` (internal/town/npc.go) learns each NPC's line offset by trial clicks and reads the UI flag; replacing it by a text verb needs the line texts per NPC and per language, which the fork's differential (step 3) must show first.
- The enable criteria of the card (one differential per verb) wait for section 3; `input.planner` stays 0.

## 5. Live checklist (not run)

Hidden instance, throwaway character with worn items below full durability, `input.planner=1` and the sandbox build listing `input.repair`: run a repair trip, expect `input=v/..` to count one verb, durability back to full, the smith window closed afterwards, character left in town with no panels open.

# Item verbs: inventory, stash and equipment actions through d2d's own item handlers (TASK-367)

> status: DESIGN plus the built verbs (equip, unequip, move, identify, drop, gold, `caps`), 2026-11-12; stage 2 and the bot wiring built in TASK-371 (user go 2026-11-12), knob still off. User decision 2026-11-12 (popup): go for item verbs. Nothing here ran against a live game; the fork slice is tested headless. Tags: V = verified (how is said), S = secondary (read in code or docs), U = unverified. Author: agent itemverbs (Claude).

## 1. Idea in four lines

- The click path (`internal/leveler`, `internal/town`, `internal/hand`) moves an item by driving the cursor: open a panel, click the item, click the target. Every step is a timing and geometry risk (panel not open, wrong rectangle, item on the cursor after a failure).
- d2d already processes an item move as server commands: `ToCursor`, `ToBody`, `ToGrid`, `ToBelt`, `Drop`, `IdentifyWith`, `StashGold`, `DropGold`, `Socket`, ... (`components/game/protocol.hpp`, handled in `components/game/world/commands.cpp`, V read). A real client packet and the click both end there.
- An item verb is a devctl line that sends the whole sequence of those commands at once, after planning it on a copy of the character with the same rule functions (`rules::can_wear`, `rules::equip`, `rules::put_in_grid`, `rules::free_spot`). The rules are the server's, not a second copy.
- The bot uses a verb when the backend advertises it in `caps` and the knob `invdesk.verbs` is on; otherwise, and after any non-rule failure, it takes the click path, which stays the oracle (and the only path on Game.exe).

Scope (user): inventory, stash, equip, identify, drop, gold. NOT combat, movement, skills, portals, shopping.

## 2. What was built (the slice)

| Part | Where | Status |
|---|---|---|
| `caps`, `item equip <id> [slot]`, `item unequip <id>` verbs | fork branch `fork-only/itemverbs` (private forge (the d2d fork repo)): `components/game/item_verbs.hpp`, `apps/d2d/devctl/verbs.hpp` | V: built and tested on the worker host; fork commit `68318c097aa6` on the private forge (branch `fork-only/itemverbs`, off `fork-only/art-clean` acadbe47, the 0.1.0 pin) |
| Fork unit test | `tests/game/test_item_verbs.cpp` (World, Sorceress and Paladin) | V: ran green on the worker host |
| Fork end-to-end test | `tests/item_verbs_d2d.py` (headless d2d, differential against the manual command sequence) | V: ran green on the worker host against the built d2d (`ctest -R item_verbs`: 2 of 2 passed; the e2e takes 14 s) |
| Go client | `internal/d2dctl/item_verbs.go` (`Caps`, `ItemEquip`, `ItemUnequip`, `ItemRefused`, adapter `ItemVerbs`) | V: `go test ./internal/d2dctl` with a stub server |
| Bot executor | `internal/invdesk/verbs.go` (`VerbEquipper`: verb first, click fallback, `LastHow`) and `Request.How` | V: `go test ./internal/invdesk` (fakes, Paladin and Sorceress) |
| Knob | `invdesk.verbs`, default 0 (`internal/env/tuning_invdesk.go`, `docs/knobs.md`) | V: knob tests |
| Stage 2 (TASK-371) | fork commit `59ae7810` (branch `fork-only/itemverbs`, private forge): `plan_move/identify/drop/gold`, `cmd drop/idwith/stashgold`, headless differential extended; bot `internal/invdesk/mover.go` (`VerbMover`, `VerbTrasher`, `HowSource`), town hooks (`Town.SetMover`, stash visit, identify batch), `cmd/d2bot/invverbs.go` wiring on the d2d backend only, `Request.How` for equip, trash, identify, stash and gold | V offline: fork unit test (2 classes), headless e2e 3 of 3 runs, `go test` of invdesk/d2dctl/town; NOT live |
| Not built | the live click-vs-verb differential (recipe and tools in section 9a), merc items, relaxing the safe moment for verbs | section 12 |

## 3. Verb set

Wire form: one devctl line, reply `<first line>\nok\n` or `err <code> <why>\n` (the devctl convention, `docs/control_channel.md` in the fork). All item verbs need the d2d flag `--item-verbs` (own gate since TASK-383; `--lab-verbs` implies it, but `--item-verbs` alone does not enable the cheat `debug` verbs). The friends release starts d2d with neither, which the legal checklist item 1 requires; without the flag `item ...` answers `err lab verbs off: ...` and `caps` does not list them.

| Verb | Reply | Server commands it sends | Stage |
|---|---|---|---|
| `caps` | `caps=state.caps,item.equip,item.unequip` | none | 1 (built) |
| `item equip <id> [slot]` | `equip id=<id> slot=<s> displaced=<id or -1> at=<c>,<r>` or `noop slot=<s> at=-1,-1` | `ToCursor`, `ToBody`, and `ToGrid` for the item that came off | 1 (built) |
| `item unequip <id>` | `unequip id=<id> slot=<s> displaced=-1 at=<c>,<r>` or `noop ...` | `ToCursor`, `ToGrid` | 1 (built) |
| `item move <id> <inv\|stash\|cube> <col> <row>` (-1 -1: first free spot) | `move id=.. at=c,r` / `noop` | `ToCursor`, `ToGrid` | 2 (built) |
| `item identify <id> [<tool-id>]` | `identify id=.. slot=0 displaced=<tool id>` | `IdentifyWith{scroll,item}` (C to S 0x27, `FUN_0054b280`) | 2 (built) |
| `item drop <id>` | `drop id=..` | `ToCursor`, `Drop` (0x17, `FUN_0054ab40`) | 2 (built) |
| `gold deposit <n>` / `gold withdraw <n>` | `gold bank=<n> carried=<n>` (the predicted balances) | `StashGold{amount,deposit}` (0x4f 0x13/0x14, `FUN_00564d50`) | 2 (built) |
| `item belt <id> [box]` | existing `cmd belt put` | exists today (`BeltPut` in `internal/d2dctl`) | done |

Capability names are exactly the verb names (`item.equip`, `item.unequip`, later `item.move`, `item.identify`, `item.drop`, `gold.stash`). `caps` is a flat comma list in one `caps=` field; a build that does not know the verb answers an error, which the Go client reads as "no capabilities" (V: `TestCapsOnForkAndStock`). A new capability never changes an old verb's reply (additive).

The commands are applied at the next tick (they go through the same queue as a click). The reply therefore says what was queued; the caller proves the end state from a fresh `items` read, exactly as the click path proves a swap. In lockstep a `step 1` applies them.

## 4. Rules, per verb, and where the original enforces them

The principle: no rule lives in the verb. The plan runs the server's functions on a copy; then the real commands run the same functions on the real state. If they disagree (another command slipped in between plan and tick) the real handler wins and the caller's read-back shows it.

### Equip (`plan_equip`)

| Check | Function (fork) | Code | Original (S unless said) |
|---|---|---|---|
| item exists in the character's list | lookup by unit id | `item` | server item table |
| backpack item only (worn: `noop` if already there, else `place`; stash/cube/belt/socketed: `place`) | location and panel test | `place` | the bot's click path takes backpack items only (`docs/inventory-actions.md`); the server's `ToCursor` accepts any listed item, so this is a deliberate narrowing, same as the bot's check |
| hand empty | `held` test | `cursor` | click path: a held item would swap |
| identified | `rules::can_wear` | `identified` | `FUN_0062eaf0` (V in `rules.hpp` comments) |
| slot allowed for the item type | `can_wear` via `ItemTypes.txt` body locations | `slot` | `FUN_0062eaf0` |
| class restriction | `type_class` against the hero class | `class` | `FUN_0062eaf0` |
| strength, dexterity, level (with requirement reductions, ethereal, socketed items) | `meets_stats`, `required_level` | `requirements` | `FUN_0062eaf0`, `FUN_0062b5b0` |
| hands: a two-hander and a shield, a second weapon for a non-Barbarian: the other hand comes off too, but only one item may come off in all | `rules::equip` (`hands_ok`, `FUN_0063dbc0`) | `twohand` | `FUN_0063dbc0`; the "only one item" limit is a `ponytail` in `rules::equip` (U against game.exe for the second-weapon case) |
| room in the backpack for the item that comes off | `free_spot` (the player's search order, `FUN_0063b490/0063b540/0063b620/0063b6e0`) then `put_in_grid` | `nospace` | the original puts the displaced item on the cursor; we never leave an item in the hand, so we refuse before moving anything |
| refused equip voice ("can't use this yet" / "impossible") | `class_cue` in the handler | none (the handler still plays it) | `FUN_00490fc0` and the other body-slot handlers via `FUN_004cb9c0` |

`slot` 0 (auto): the one slot the item fits, else the first empty one, else the first. This is the bot's `autoSlot` rule (rings, hands).

### Unequip (`plan_unequip`)

Item must be worn (`place` otherwise; a backpack item is `noop`), hand empty (`cursor`), a free backpack spot (`nospace`). The original has no further rule (a worn item may always be taken off, V in `commands.cpp`: `ToCursor` is `pick_up`). Charms, belts: taking a belt off with potions in it is the server's business (`belt_index` changes); not covered by the slice, U.

### Move (stage 2)

`ToGrid` rules: `put_in_grid` (bounds by `inv_layout` / `stash_layout` / `cube_layout`, overlap with at most one item which is swapped to the hand, cube never into itself). The verb adds the same "never leave an item in the hand" rule: a swap target means `nospace`-or-`overlap` refusal before moving. Stash precondition: the server code does not check that the stash panel is open or the hero is near the chest (`ToGrid` with panel 5 just uses the layout, V read in `commands.cpp`); the stash UI opens only near the chest (`tick.cpp`, "Close enough to the stash", 2 cells, a `ponytail`). So a stash move verb must itself require "in town and within the stash object's operate range" or the bot would gain a power the player lacks. Rule (user go 2026-11-12: strict): the verb refuses with `far` unless the chest is the hero's current interaction target and within the same 2 cell test; the click path walks there as before. The codex cross-check agrees in kind: 0x19 (take out of a grid) lets an idle player lift only inventory-page items unless a stash or cube interaction is open (S, section 15). The numeric range stays U.

### Identify (stage 2)

Server handler: `IdentifyWith` (`FUN_00561ed0`): a Scroll of Identify (`isc`) is used up; a Tome (`ibk`) loses one charge (stat 0x46); the item must exist and be unidentified; anything else is silently ignored. The verb checks these four and answers `identified` / `notool` / `item`; it does not choose the tool unless asked (`with` omitted: prefer the tome with charges, else a loose scroll, the click path's rule `identifyOne`). **Packet order V (disassembly read 2026-11-12, game.exe 1.14d, offline):** `FUN_0054b280` loads `ecx = [packet+1]` and `ebx = [packet+5]`, looks both up as items on the player (`FUN_00549220`), then calls `FUN_00561ed0(player, [packet+1], [packet+5], &out)`; inside, the first id is the target item and the second is checked with `FUN_00629bb0(item, 0x16)` (item type 0x16 = the scroll family, which includes the tome), i.e. 0x27 is `{target item u32@1, tool u32@5}`. That agrees with the two outside sources and contradicts the note in `docs/probes/fid/fid-identify.md` (scroll first); that note should be corrected by its owner. d2d's `IdentifyWith{scroll,item}` is a struct, not a wire layout; no raw 0x27 path exists in d2d (`components/net/d2gs` only names it), so a future raw packet must write the item first. Cain's identify-all (0x34, `FUN_00578460`, 100 gold an item, free once the quest is done) is an NPC dialog command (`cmd::Identify`), not an item verb: the bot reaches it through the dialog and keeps doing so. Gold and quest state are the server's.

### Drop (stage 2)

`Drop` (0x17, `FUN_0054ab40`): the item in hand onto the ground at the hero's feet, refused while dead. The verb is `ToCursor` + `Drop`; refusals: `item`, `cursor`, `dead`. Landing spot: `loot.place` at the hero (S; the click path picks a free cell beside the hero, so the two paths differ in the exact tile; the differential compares "on the ground within 2 cells" not the tile).

### Gold (stage 3)

`StashGold` (0x4f 0x13/0x14, `FUN_00564d50`): deposit up to the stash cap (2,500,000 total); withdraw up to carried cap (clvl x 10,000). The verb forwards the amount and reports both balances after clamping; it adds no limit of its own. Needs the same stash proximity as a move. `DropGold` stays out of scope.

### Mercenary items

The merc panel is a different command (`MercItem`, 0x61, `FUN_0054d430`: slot 0 puts the held item on the merc, slot 1..10 takes it into the hand; `merc_slot` rules `FUN_00553380`). The hero verbs do not touch it. Stage 4 (optional): `item merc-equip <id>` / `item merc-unequip <slot>` with the same plan-on-a-copy approach; until then `Request.Target` "merc" stays refused (`docs/research/merc-gear.md`).

## 5. Error codes

One vocabulary for both paths. The verb answers `err <code> <why>`; the Go client maps it to `d2dctl.ItemRefused{Code, Why}`; the bot's click-path pre-check already produces the same reasons (`internal/leveler/playerequip.go`: `ErrNoItem` = `item`, `ErrNoSlot` = `slot`, `ErrOffHand` = `twohand`, `cand.Why` strings for class/level/stat/identified = `class`, `requirements`, `identified`).

| Code | Meaning | Falls back to click? |
|---|---|---|
| `item` | no such unit id | no |
| `cursor` | the hand is not empty | no (retry later: the bot postpones) |
| `place` | wrong place (not a backpack item / not worn / stash needs the chest) | no |
| `slot` | item does not fit that slot | no |
| `identified`, `class`, `requirements`, `twohand`, `nospace` | rule refusals | no |
| `far`, `notool`, `dead` (stage 2+) | state preconditions | no |
| `lab verbs off`, `unknown verb`, `not in a game`, transport error, reply timeout | the verb did not run | yes, and the capability is dropped for a minute |

A refusal is final: the click path would fail for the same reason, so retrying it would only hide a disagreement between the bot's pre-check and the server. Such a disagreement (the bot says ok, the server refuses) is logged at warn with both texts and is the first thing the conformance suite looks for.

## 6. Idempotency and failure

- Equip of an item already on the requested slot, unequip of an item already in the backpack: `noop`, `ok`, nothing sent. Repeating a verb after a lost reply is therefore safe.
- Plan first, move second: a refusal moves nothing; success is one queue of at most three commands in one tick, so the cursor is never occupied between ticks (V: unit test asserts the hand is empty after every plan).
- If another player command is queued between the plan and the tick (the devctl is single-client, so only the bot itself), the real handlers still apply the rules; the worst case is a refusal visible only in the read-back. The bot's `Verify` hook (wired in stage 2) reads a fresh snapshot after the reply and fails the request with "the item verb answered ok but the item is not worn" instead of falling back (the item may have moved).
- Item on the cursor at the start: `cursor`. The existing click path's recovery (`hand.DropCursor`) is unchanged.

## 7. Interplay with the player-command queue

Unchanged. The verb replaces only the body of `Equipper.PlayerEquip` (`internal/invdesk/desk.go` default case). Queue order, `invdesk.priority`, the safe-moment predicate (`Runner.safeMoment`), the priority ladder (safety first, then player commands) and the `errPlayer` preemption all stay as in `docs/inventory-actions.md`. What changes:

- A verb does not walk or open panels, so it is safe in more moments than a click: a fight nearby no longer needs to postpone an equip. Stage 3 may relax `leveler.busy` for the verb path; the first release keeps the same safe moment so the two paths are comparable (and the equip still takes game time: the server applies it on a tick).
- Ownership: the Desk worker is the only caller; `VerbEquipper` serialises through the Desk's single `RunPending`, and `LastHow` is mutex-protected.
- The verb is player-commanded work only (docs/gear-policy.md: the bot makes no equipment decisions); the knob does not change which equips happen.

## 8. Bot executor interface

```
invdesk.ItemVerbs  { Caps(ctx) ([]string, error); ItemEquip(ctx, id, slot) (noop bool, err error) }   // d2dctl.ItemVerbs adapts a client
invdesk.VerbEquipper{ Click Equipper; Verbs ItemVerbs; On func() bool; Verify func(ctx,id,body) error }
   implements Equipper (CheckPlayerEquip always from Click) and Howser (LastHow: "verb" | "click")
```

- `PlayerEquip`: knob off, no `Verbs`, or no `item.equip` in `caps` (read once a minute, a transport error is not cached) means the click path. Otherwise the verb; a rule refusal (`RuleRefusal()` on the error) is returned as the request's failure; any other error drops the capability for a minute and runs the click path; a context error is returned as is (the Desk postpones).
- Telemetry: `Request.How` (`json:"how,omitempty"`, additive, empty for old files and other kinds) and the done line `equip X -> slot done, slot pinned (how=verb)`. Per-action counters for the Why/telemetry page are a stage 3 item.
- Wiring (stage 2): `wireInv` gets the d2dctl client from the d2d backend (`cmd/d2bot`), builds `invdesk.NewVerbEquipper(lv, d2dctl.ItemVerbs{C: c}, func() bool { return tune.Invdesk.Verbs != 0 })` and passes it to `Desk.SetEquipper`. On Game.exe there is no client, so `Verbs` is nil.
- The same pattern repeats per kind: a `VerbTrasher`, a `VerbIdentifier` wrapping `Services.PlayerIdentifyBatch`.

## 9. Conformance: differential test plan

Goal: for every verb the verb path and the click path leave identical game state, in both directions, and the fork keeps matching Game.exe where the oracle can say.

1. **Headless unit (built, V).** `test_item_verbs` builds a real `World`, applies the plan's commands through `World::tick` and asserts the end state equals what the plan said (slot, displaced item's cell, hand empty), for a Sorceress and a Paladin, with refusals (identified, requirements, slot, class, twohand, nospace, cursor) leaving the item list unchanged.
2. **Headless e2e, verb vs manual sequence (built, V).** `tests/item_verbs_d2d.py` starts two d2d processes with the same seed, gives the same items (`debug give`), and drives one with `item equip/unequip` and the other with `cmd hand` / `cmd body` / `cmd grid` (the exact command a click sends, placed where the verb reported). It compares the final `items` listing by code, location, slot, panel and cell (unit ids differ by construction).
3. **Click path vs verb on one state (live checklist, U).** Per action sequence (equip into empty slot, swap, two-hander over weapon and shield, ring into either finger, unequip, identify with scroll and with tome, drop, stash move, gold deposit and withdraw): fresh character, same `--seed`, lockstep, `debug give` the same items; run A with the bot's click executor (`invdesk.verbs=0`), run B with the verb (`=1`); after each action compare `items` (position, equipment, panel), `state` gold, scroll and tome counts, ground items (drop within 2 cells). Then swap the order (verb first, click first) to catch state-dependent differences. Both directions: verb result replayed by click and click result replayed by verb must give the same state.
4. **Original as oracle (S).** For the rules the fork marks `ponytail` (second weapon displacement, stash range, belt removal), run the same sequence on Game.exe through the existing emulator oracle (`docs/d2-1.14d-memory.md`, `tools/emu`) and record the verdict in `docs/probes/`; a difference is a fork bug (upstream-pr candidate), not a verb bug.
5. **Regression guard in CI.** The Go fakes tests (section 2) fail if the executor stops falling back; the fork tests run in the fork's ctest set; a nightly job runs 3 and 2 on the worker host.

All of 3 and 4 need the game or the oracle: ready-to-run checklist, not run here.

## 9a. Live click-vs-verb differential: recipe and tools (prepared, not run live)

NOT the sandbox container and no deployment: use a throwaway hero on a hidden instance or a separate throwaway container and volume.

1. Offline, already runnable: `python3 -I tools/itemverbs/diff_d2d.py <fork d2d> [--report FILE]` on the worker host with `D2_MPQ_DIR` set (a throwaway headless d2d pair, own HOME and socket). V: 3 of 3 runs matched (equip into an empty slot, swap with the starting weapon, unequip, pack move, identify with a scroll, drop, nine refusals that move nothing).
2. Click path vs verb (hidden instance, throwaway hero, same `--seed`, lockstep, `d2cheat`-free setup by `debug give` only, declared in `docs/probes/test-characters.md`): run the bot's own click executor with `invdesk.verbs=0` for the sequence in section 9 item 3, `d2dctl.Items` dump after each action into `click-N.items`; restart the same hero state, run with `invdesk.verbs=1`, dump into `verb-N.items`; then `python3 -I tools/itemverbs/compare_items.py click-N.items verb-N.items` per step (exit 0 = same item code, location, slot, panel and cell). Run both orders. Gold and scroll counts are read from the `state` line pasted into the dump.
3. Stash and gold steps need the hero at the chest: the click run walks there itself; the verb run runs the same visit (the town opens the stash first, then asks the mover), so the visit and the hero position are the same in both.
4. Record the result in `docs/probes/` with `writes after start: 0` (no gameplay write after the start; the verbs are not memory writes).

## 10. Crutch rule and decision text (DRAFT for the user; not written to `docs/adr` or `docs/decisions.md`)

Why this is not a memory write: an item verb writes no game memory and no gameplay state outside the server's handlers; it submits the commands a client would submit. ADR 0014's crutch rule is about memory writes. Two things still need saying: the bot may use a verb in a measured run (the result is what a real client packet produces, with the same rules), and the verb must never offer an action the player cannot do (no free identify, no gold beyond limits, no stash from afar).

FINAL DRAFT ADR 0026 (for the user's grill session; the user's go of 2026-11-12 to turn the verbs on for the sandbox hero is the decision it records):

> **0026: Item verbs (d.idle fork)**
> Status: accepted for the sandbox hero (knob default stays off). Date: 2026-11-12. Decision-makers: user.
> Context: inventory, stash, equip, identify and drop are driven by cursor clicks; timing and geometry cause failures (items left on the cursor, panels not open).
> Decision: the d.idle fork may offer `item.*` and `gold.*` devctl verbs behind the d2d flag `--item-verbs` (off by default; not a cheat gate). A verb plans on a copy with the server's own rule functions and then submits the same server commands a client packet produces. A verb enforces every rule the original enforces and adds none; where the original needs a precondition (stash range, tool, identified, hands) the verb refuses with a named code. Verbs write no memory. The bot uses a verb only when the backend lists it in `caps` and the knob `invdesk.verbs` is on; the click path stays as fallback, as oracle for the conformance suite, and as the only path on Game.exe. Per action the bot records `how=verb|click`.
> Scope: inventory, stash, equipment, identify, drop, gold. Not combat, movement, skills, portals, shopping.
> Consequences: two code paths must agree (differential suite of `docs/design/item-verbs.md` section 9 is required before the knob defaults on); the friends release starts d2d without `--item-verbs`, so it keeps the click path; the sandbox hero runs with the flag and the knob `invdesk.verbs` on by a local override only; a measured run may use verbs (they are not writes), and its record carries the `how` split.

DRAFT `docs/decisions.md` lines:
- "Item verbs (user, 2026-11-12, popup): the fork gets `item.*` / `gold.*` devctl verbs that run the server's own item handlers; bot uses them behind `invdesk.verbs` (default off) with the click path as fallback and oracle. Scope inventory, stash, equip, identify, drop, gold. ADR 0026 (draft in docs/design/item-verbs.md)."
- Current truth bullet: "Item verbs: lab verbs, not memory writes; allowed in measured runs; the click path remains required and tested."

Also for `docs/cheats.md` (owner's call): one line that item verbs are not cheats under ADR 0014 because they write no memory, and that the legal checklist's "lab verbs off by default" covers them.

## 11. Risks

| Risk | Mitigation |
|---|---|
| Two code paths rot; the click path is no longer exercised | the knob stays off by default; the conformance suite runs both; the click executor is the fallback and runs whenever the cap is missing, so Game.exe runs keep it alive; a nightly job alternates the knob |
| Fidelity: verb results differ from the original (the fork's `ponytail` rules) | the verb adds no rule; differences are fork-rule bugs found by plan item 4 and fixed upstream |
| Item on the cursor edge cases (leftover hand, a drop that failed) | the plan refuses when the hand is not empty and never leaves an item in the hand; the click path's cursor recovery is unchanged |
| Verb gives powers a player lacks (stash from afar, identify without a tool) | per-verb preconditions in section 4; any new verb needs its precondition list reviewed against the original before it ships |
| Mercenary items | out of the slice; separate command and rules (section 4) |
| Reply is "queued", not "done" | `Verify` read-back; `noop` idempotency; lockstep `step 1` in tests |
| Lab verbs are off in the friends release, so the feature never reaches friends | by design for now; shipping it means a decision to allow a non-writing verb group outside `--lab-verbs` (open question 2) |
| `cmd` queue ordering with the bot's own other commands | single devctl client; the Desk is the only caller during a request |
| Capability cache stale after a d2d restart | one-minute TTL and a drop on any non-rule failure |

## 12. Staged plan and estimates

| Stage | Content | Estimate |
|---|---|---|
| 1 (done) | `caps`, `item equip`, `item unequip`, unit and e2e tests, Go client, executor with fakes, knob, doc | this task |
| 2 (done, TASK-371) | wire `VerbEquipper` in `cmd/d2bot` with `Verify`; `item identify`, `item drop`, `item move` (backpack and stash with the range rule); matching `VerbTrasher`/identify wrappers; fork tests | 1 to 1.5 days |
| 3 | `gold deposit/withdraw`, per-action `how` counters on the Why page, relax the safe moment for verbs, conformance runs 3 (live, both directions) | 1 day plus live time |
| 4 | oracle runs for the `ponytail` rules, decision on the default of the knob, merc items | 1 day plus the user's decision |

## 13. Open questions for the user

1. Knob default: stay off until a live differential run (section 9 item 3) is green, or turn on for the sandbox only after stage 2?
2. Gate: keep item verbs under `--lab-verbs` (friends release keeps the click path), or move the non-writing item verbs to an always-on group so the release can use them?
3. Stash range: the fork's 2-cell test is a `ponytail`; is a verb refusing with `far` acceptable, or should the stash verbs wait for the original's range to be decomposed?
4. ADR 0026 and the two decision lines: yours to write (agents may not).

## 14. Dead ends and notes

- A new `cmd::EquipItem` command in the protocol variant would make the move atomic on the server, but it touches the wire codec, replication and the net game (`protocol.hpp` wire section); sequencing the three existing commands in one `town.send` batch gives the same atomicity (one tick) without a protocol change. Revisit only if a tick boundary ever splits a batch.
- Applying the plan directly to `actor->character.items` from the verb handler would be synchronous but would bypass the handler's sounds, class voice and (later) replication; rejected.
- The `cmd belt put` verb is the precedent (V in the fork docs, `BeltPut`): a verb that sends `ToCursor` and `ToBelt`. Item verbs generalise it with a plan step so that a refusal leaves the item where it was (`belt put` onto an occupied box leaves the displaced item on the cursor).

## 15. References (read-only cross-checks, nothing copied)

- Codex specs of the 1.14d client-to-server item packets (MIT for specs; summarised, with licence tags, in `docs/research/item-packets-2026-11-12.md` on branch agent/mine1, queued #630): used only to cross-check the rule list: 0x17 drop needs a cursor item and drops at the feet; 0x18 page rules and the grid fit test; 0x19 an idle player may lift only inventory-page items unless a stash or cube interaction is open (supports the strict stash rule); 0x1A bodyloc 1..10 and a silent refusal on level/strength/dexterity (matches `can_wear`); 0x4F buttons 0x13/0x14 stash gold (`FUN_00564d50`, 2.5M stash cap, partial deposit, carry cap clvl x 10,000: matches `plan_gold`); 0x50 drop gold (not built). Tag S until a probe confirms.
- The 0x27 field order is settled V by disassembly (section 4, Identify).

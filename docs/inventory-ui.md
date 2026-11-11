# Inventory page on the watch screen (TASK-212, invui)

> status: the item action menu (equip, sell mark, trash, queue states, field equips) is in [inventory-actions.md](inventory-actions.md); where it differs from the equip rules below (equip is no longer town only), that doc wins.

Page `/watch/inventory` (link in the watch header), data `GET /api/inventory`, writes `POST /api/inventory/{mark,equip,unpin}` on the answer listener. Tags: V verified by the offline tests named, U unverified (needs a live run or game tables).

## What it shows
Equipped slots, the 10x4 backpack, belt and 6x8 stash as grids, with name, quality colour, props, item level and a "why" line (the loot judge's verdict, `loot.Verdict.String()`; U live that the text reads well). Items come from the existing `status.InventoryData` (`cmd/d2bot/display.go` `invSource`), so names and sizes match the old dashboard panel. The stash lists only what the snapshot lists (U: whether memory shows the stash while it is closed, see docs/inventory.md). Each item has an `advice` field, empty today: the hook for the `upgradeadvisor` card (`invdesk.Config.Advice`). The bot fills it from a cache with what it would have worn (docs/gear-policy.md: the player decides equipment, the bot only advises), and `View.Prompt` carries the onboarding line.

## Marks (sell, keep)
- `sell`: sold at the next vendor visit (`town.sellableIf` checks the mark before the pickit). No extra trip is raised. Only backpack items can be marked; quest items, keys, tomes, the Cube and start items are refused (`town.NeverSellReason`, reason line in the reply).
- `keep`: never sold or dropped by the town (`sellableIf` skips it, `sellProtected` is true). Allowed in any container.
- Unmarked items follow the existing rules unchanged. Marks persist in `inventory-ui.json` in the bot's state dir and are keyed by item id plus class/quality/ilvl, so a reused id of another item does not inherit one (U live: ids across game restarts).

## Equip and pins (pins are legacy)
- `POST equip {id, slot}` queues "wear backpack item id in slot". Refused at once, with a reason line (HTTP 409 + code): not in town (`town`), item not in the backpack (`place`), unknown slot, class/level/strength/dexterity/unidentified (`requirements`, from `leveler.Candidate`, the rule the bot's own gear logic uses; a low-quality weapon is allowed), slot mismatch, two-hander with an off-hand item worn.
- The runner performs the queue at the gear step in town (`bot.PlayerGear`, `gearStep`), ignoring the check interval, through the leveler's verified click path (`Leveler.PlayerEquip` -> `doSwapRetry`). A fight nearby postpones it; a failure is shown in the page and does not pin. No memory writes.
- DEPRECATED (2026-11-11, card nopins): pins are shown nowhere any more (no PIN badge, legend chip, Unpin button, menu entry or tooltip badge). Under the player-only gear policy (resource.gearDecisions=player) the bot never equips or swaps, so a pinned slot carries no information. The API keeps `pins` in the view (deprecated, read-only; may be empty) and `POST /api/inventory/unpin` (deprecated, still works) for old clients and manifests; the golden fixtures keep them. The legacy bot-gear path (gearDecisions=bot, drills and soaks) still honours `Desk.Pinned`.
- History: a successful equip pins the slot. Pins are per slot, persist and have NO expiry and no "clearly worse" override (user decision 2026-11-09). Only `POST unpin` ends one. A pinned slot is skipped by `town.foundGear`, `equipTodo`, and `leveler` (`Options.Pinned`: the worn item scores `pinnedScore`, so `Equip` and `Shop` plan nothing against it). The bot may only suggest upgrades (card `upgradeadvisor`, TASK-213).

## Safety and auth
POST routes use `policyGate`: origin list, token or passcode cookie (the answer listener's gate), JSON only, small body, unknown fields refused. Replies never include internal error text.

## Tests
V (run here): `go test ./internal/invdesk ./internal/watch -run Inventory` (marks, refusals, pins, persistence, queue, Paladin and Sorceress tables, route gates). U (compile, vet, skip without game tables, so not run here): `internal/town/invmarks_test.go`, `internal/leveler/playerequip_test.go` (Sorceress and Paladin).

## Open choices (safe defaults taken)
1. Equip outside town is refused (default); a field equip at a calm moment is possible later since `Leveler.PlayerEquip` already refuses when monsters are near.
2. Sell marks only on backpack items; stash items must be moved by hand.
3. A rejected request is not retried; a postponed one (fight) is.
4. Sold items are not re-offered; no "undo".

## Live checklist (nothing here ran live)
1. Open `/watch/inventory`: slots and grids match the game's inventory panel.
2. Mark a magic ring for sale, keep a white item in a full pack; run a town visit: ring sold, kept item stays.
3. In town request a better armor from the pack: expect "player equip ... done, slot pinned" in the log, item worn, cursor empty; then drop a better piece into the pack and visit town again: it stays unworn. Unpin, visit again: the bot may swap.
4. Request from the field: refusal "not in town". Request a Paladin-only item on another class: refusal with the requirement.
5. Restart the bot: marks and pins are still there.

## Merc doll (TASK-287, stage A of docs/research/merc-gear.md; read only)
Tags: V verified offline (test named), U unverified (nothing was run live or in a real browser).
- `GET /api/inventory` is schema 4 (3 is the action menu): an additive, optional `merc` block (`invdesk.MercView`: `state` none, ready, dead or unknown; `owned`, `alive`, `class`, `kind`, `level`, `str`, `dex`, `lifePct`, `slots` (the slots this kind can use), `equipment` keyed `mhead`, `mtorso`, `mrhand`, `mlhand`, `note`). A schema 2 reader ignores it; a reply without it (a backend that cannot read a merc) hides the card (V `merc_dom.js` state `absent`, `TestMercFixturesDecodeIntoView`).
- Files: `internal/watch/web/inventory-merc.js` and `.css`, loaded by the page after `inventory.js`; it follows the `d2:inventory` event and does not edit `inventory.js`. The card sits under the hero doll (`#merc-doll`), a `details` fold that is open while a merc is hired. The doll is a `d-doll` with `data-owner="merc"` and four `d-slot2` (`data-slot` mhead, mtorso, mrhand, mlhand); a slot the kind cannot use is `data-state="na"` with an n/a badge (Desert Guard, Rogue and Barbarian have no shield). Game art is the optional layer of the hero doll: item icons `item-<art>` and the ghost keys `slot-helm`, `slot-armor`, `slot-weapon` (V golden `desert_art`; a failing image reverts to the drawn slot; the merc panel frame key is not built).
- No actions: no equip, take-off, mark or pin for the merc (stage B and C, `target: merc` in the action queue of the player-side card); the bot makes no merc equipment decision (docs/gear-policy.md). Merc items get no marks, pins or advice (V `TestViewMercBlock`).
- Tooltips: merc items carry the base-type facts, no d2d tooltip text yet (the fork dump prints only the first text line; a verb that prints the merc item text is a follow-up, U). The slot `title` shows name, durability and item level.
- Goldens: `internal/watch/testdata/inventory_merc_*.json` (views) and `testdata/mercdoll/merc_dom_*.txt` (DOM); `UPDATE=1 node testdata/mercdoll/merc_dom.js web/inventory-merc.js` rewrites the latter. Live by eye (docs/live-checks.md, TASK-287): the card in a real browser at phone and desktop width, light and dark, with a hired merc from the fork build.

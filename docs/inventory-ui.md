# Inventory page on the watch screen (TASK-212, invui)

Page `/watch/inventory` (link in the watch header), data `GET /api/inventory`, writes `POST /api/inventory/{mark,equip,unpin}` on the answer listener. Tags: V verified by the offline tests named, U unverified (needs a live run or game tables).

## What it shows
Equipped slots, the 10x4 backpack, belt and 6x8 stash as grids, with name, quality colour, props, item level and a "why" line (the loot judge's verdict, `loot.Verdict.String()`; U live that the text reads well). Items come from the existing `status.InventoryData` (`cmd/d2bot/display.go` `invSource`), so names and sizes match the old dashboard panel. The stash lists only what the snapshot lists (U: whether memory shows the stash while it is closed, see docs/inventory.md). Each item has an `advice` field, empty today: the hook for the `upgradeadvisor` card (`invdesk.Config.Advice`). The bot fills it from a cache with what it would have worn (docs/gear-policy.md: the player decides equipment, the bot only advises), and `View.Prompt` carries the onboarding line.

## Marks (sell, keep)
- `sell`: sold at the next vendor visit (`town.sellableIf` checks the mark before the pickit). No extra trip is raised. Only backpack items can be marked; quest items, keys, tomes, the Cube and start items are refused (`town.NeverSellReason`, reason line in the reply).
- `keep`: never sold or dropped by the town (`sellableIf` skips it, `sellProtected` is true). Allowed in any container.
- Unmarked items follow the existing rules unchanged. Marks persist in `inventory-ui.json` in the bot's state dir and are keyed by item id plus class/quality/ilvl, so a reused id of another item does not inherit one (U live: ids across game restarts).

## Equip and pins
- `POST equip {id, slot}` queues "wear backpack item id in slot". Refused at once, with a reason line (HTTP 409 + code): not in town (`town`), item not in the backpack (`place`), unknown slot, class/level/strength/dexterity/unidentified (`requirements`, from `leveler.Candidate`, the rule the bot's own gear logic uses; a low-quality weapon is allowed), slot mismatch, two-hander with an off-hand item worn.
- The runner performs the queue at the gear step in town (`bot.PlayerGear`, `gearStep`), ignoring the check interval, through the leveler's verified click path (`Leveler.PlayerEquip` -> `doSwapRetry`). A fight nearby postpones it; a failure is shown in the page and does not pin. No memory writes.
- A successful equip pins the slot. Pins are per slot, persist, show on the page and have NO expiry and no "clearly worse" override (user decision 2026-11-09). Only `POST unpin` ends one. A pinned slot is skipped by `town.foundGear`, `equipTodo`, and `leveler` (`Options.Pinned`: the worn item scores `pinnedScore`, so `Equip` and `Shop` plan nothing against it). The bot may only suggest upgrades (card `upgradeadvisor`, TASK-213).

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

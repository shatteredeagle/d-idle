# Inventory item actions (TASK-282, itemactions)

Click, tap or press Enter on an item on `/watch/inventory` (backpack, stash, belt or a worn slot): a menu offers Equip, Mark for sale, Keep, Trash and (worn pinned slot) Unpin. A choice QUEUES a request; the bot performs it at the next open opportunity. A phone gets the same menu as a bottom sheet. Tags: V verified by the offline tests named, S secondary, U unverified. NOTHING here ran live.

## Request model (`internal/invdesk`)
- `Request{id, kind equip|trash, slot, target, name, state, reason, at, print}`. States: `queued` -> `doing` -> `done` | `failed` (reason); `cancelled` only from `queued`. `target` is reserved for the mercenary stage (docs/research/merc-gear.md): "" or "hero" today, the watch route answers 409 code `target` for anything else.
- The queue is persisted in `inventory-ui.json` (`queue`, `recent`), at most `MaxQueue` (8) pending, further requests refused with code `full`. A `doing` entry found at load is reported `failed` ("interrupted by a restart") and never retried: the item may have moved. V: `TestCancelBoundAndPersistence`, `TestDoingCannotBeCancelledAndRestartFailsIt`.
- Never undone: a done equip keeps its pin (no expiry), a done trash stays dropped; the bot never re-equips or picks the item up. A queued equip of an item is replaced by a trash of it; equip of a trash-queued item is refused.
- Callbacks (Snap, SellBlock, TrashBlock, Equipper, Trasher) always run with the Desk mutex released (TASK-278).

## Open opportunity (`internal/bot/playeraction.go`, U live)
`playerActionStep` runs in `runGoal` right after the points step, same predicate family: no town trip running, the hero is not walking or running (`Player.Mode` 2, 3, 6; the mode numbers are U), and either in town or in the field with `hostileNear(ClearRadius) == 0`. The click path checks again (`leveler.busy`: a live monster within 25 subtiles postpones, the request goes back to `queued`). After a postponement the step waits `playerActionGap` (10 s). The Why line shows `player_items` ("Doing what you asked on the inventory page"). V: `TestPlayerActionsRunInTownAndWaitForTheFieldFight` (fake runner).

## Equip
Slot is chosen by the page from `canWear` (one slot: plain "Equip"; two, such as both ring fingers: "Equip (left ring)" and "Equip (right ring)"). The API also takes an empty slot: the only fitting slot, else the empty one, else code `choose`. Class, level, strength, dexterity and identification checks are the existing `Leveler.CheckPlayerEquip`. Only backpack items. V: `TestAutoSlotRings` (ring fingers), `TestEquipQueuesInTheFieldAndFailureDoesNotPin`.

## Mark for sale, Keep
Unchanged (docs/inventory-ui.md); one click from the menu.

## Trash (new primitive)
- UI: the first click turns the entry into "Confirm: trash it" (plus "Keep it"); only the second click POSTs `{id, confirm:true}`; the server refuses without `confirm` (code `confirm`).
- Refused (reason shown in the menu): marked keep (`keep`, also re-checked when the request runs), worn items pinned or not (`equipped`: take it off first), stash, belt and cube items (`place`: move to the backpack), quest items, keys, tomes and the start kit (`never`, `town.NeverSellReason`). V: `TestTrashRefusals` (Paladin and Sorceress rigs).
- Click path (`leveler.PlayerTrash`): open the pack, `hand.Take` the item, `hand.DropCursor` (panels closed, left click on a free playfield cell beside the hero, up to `town.DropTries` cells), then prove from memory that the item left the pack and the cursor is empty. If no ground click worked, `DropCursor` puts it back in the pack and the request fails (`ErrNoRoomToDrop`); the item never stays on the cursor. Real input only, no memory writes. Test `TestPlayerTrash*` (Sorceress and Paladin, sim world) compiles and vets but SKIPS without game tables, so it did not run here (U).
- Ignore: a done trash records id plus fingerprint (class/quality/ilvl) and the hero position in the state file; `Desk.Ignored` feeds `loot.Looter.SetIgnore`, and the pickup skips the item without a report line. An item with the same fingerprint within 12 subtiles of the drop point also matches, because a dropped item may get a new unit id (U live). Lifetime `TrashTTL` (2 h) and at most 32 entries. V: `TestTrashRunsDropsAndIgnores`; the looter test `TestPickUpLeavesTrashedItems` skips without game tables (U).

## API (`watch.inventory` schema 3, additive)
`POST /api/inventory/trash {id, confirm}`, `POST /api/inventory/cancel {id, kind}`; both use the policy gate (origin list, token or passcode cookie, JSON, small body, unknown fields refused) like mark/equip/unpin. `GET /api/inventory` schema 3 adds item fields `noWear`, `noSell`, `noTrash` (reasons) and request fields `kind`, `target`, `print`, states `doing` and `cancelled`. Nothing removed or renamed; equip is no longer refused for "not in town" (code `town` is gone). Goldens: `internal/invdesk/testdata/view_schema3_{Paladin,Sorceress}.json` plus a test that every schema 2 key is still there and an old state file loads.

## UI (`web/inventory-actions.js`, `.css`)
Own module, listens to the `d2:inventory` event and delegates clicks, so `inventory.js` is unchanged apart from one hint text and the queue card heading. Menu is `role=menu`, entries `role=menuitem`, refused entries stay focusable with `aria-disabled` and a visible reason; arrows, Home, End, Tab cycle, Escape closes and returns focus; every `focus()` passes `preventScroll`; the popover is `position: fixed` placed by a transform, the sheet layout is a `data-mode` flag (no media query). The page's grid refreshes on its own 4 s poll; the menu and the queue card refresh at once. V: `TestItemActionsDOMGoldens` (node: 14 menu states for both classes, trash confirm, queue card, keyboard, sheet, poll stability), `TestItemActionsJSNeverScrollsOrInjects`, `go run ./tools/uilint` clean.
Extension point for drag and drop onto the doll (later pass): doll buttons already carry `data-slot` and `data-key="slot-<slot>"`; a drop handler calls `window.d2InvActions.equip(itemId, slot)`.

## Live checks (nothing ran live; add to docs/live-checks.md when the card closes)
1. Equip a ring from the menu in town: left and right ring entries show when both are free; the ring is worn, the slot shows PIN, queue shows queued, doing, done.
2. Equip from the field with no monster near: waits while a fight is on, runs after; Why line shows `player_items`. Check `Player.Mode` values while walking (2, 3, 6) and standing.
3. Trash a junk item in the field: the item lies beside the hero, the pack no longer has it, the cursor is empty, the looter does not take it back across a full area pass; check the unit id of the dropped item equals the pack id.
4. Trash with no reachable free cell (crowded corner): the request fails and the item is back in the pack.
5. Cancel a queued request from the menu and the queue card; restart the bot with a queued request (still queued) and mid-action (reported failed).
6. Phone: tap an item, the sheet opens, no scroll jump; keyboard: Enter, arrows, Escape.
7. Trash while a Keep mark exists is refused; quest item refused.

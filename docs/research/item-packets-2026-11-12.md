# Item packets and the server's validation of item actions, 1.14d (2026-11-12)

> status: research reference (card mine1) for the item-verbs design (docs/design/item-verbs.md on branch agent/itemverbs). Facts and addresses are recorded with their source; nothing is copied from the repos named. Classic LoD 1.14d only (not D2R). Little-endian, one id byte first.

## Sources and tags

- **X** codex specs, https://github.com/Nugwa628/d2-114d-codex @ e4497e1d (MIT for specs; decompiler output in it is unlicensed): `specs/sim/client-messages.tsv`, `specs/sim/server-messages.tsv`, `specs/items/inventory-moves.md`, `specs/items/inventory.md` section 5, `specs/world/vendors-2.md` section 10, `specs/world/cube.md`. Written from the game.exe decompilation with a "confirmed" column and recorded runs. Tag S (I read the specs; I did not read the decompilation).
- **P** community packet data, https://github.com/blizzhackers/Diablo2PacketsData @ addebc2 (2026-01-03; NO licence file, read only): `src/data/1.14d/client2gs.json` (91 packets) and `gs2client.json` (131). Tag S.
- **N** https://github.com/MephisTools/diablo2-protocol @ 173e557 (2019, MIT) `data/1.14/d2gs.json`: field NAMES only (itemId, x, y); sizes absent. https://github.com/jaenster/libd2 @ 00004e6 (MIT) `packages/net/src/cs.zig`: the client-to-server size table (clean-room; "checked against recovered layouts, not a live server" per its docs/VERIFICATION.md). Tag S.
- **O** our own docs. A handler address that our docs already tag V is marked V(O).

Checks I did: X, P and libd2's size table agree on the size of every item packet 0x16-0x2A, 0x32-0x37, 0x4C, 0x4F, 0x50, 0x60-0x63 (S, compared field by field for 0x16-0x2A and 0x4F, 0x50, 0x63). The handler addresses of X match the FUN_ addresses our docs use for 0x20 (0054b1e0), 0x27 (0054b280), 0x63 (0054d520), 0x41 (0054c0e0), 0x49 (0054c5d0) and the stash-gold command (00564d50): V(O). Everything else in this file is S until a probe confirms it.

## How the server treats a client message

S (X intents-events.md, rules 2.4 and the classifier): a message is dropped before any handler when its size does not equal the size table entry (handler result 3: dropped silently in local mode; in network mode 3 waits for more bytes and the invalid-id result 4 bans the sender). Handler result codes: 0 done, 1 refused (target missing, out of range, wrong state), 2 invalid field, 3 malformed. "None of them has a consequence in 1.14d" except 3 and 4 dropping the message: a refused item action produces NO error message to the client; the client just sees nothing change (or, for some, a "can't do that" 0x5A, see below). Consequence for a headless client: success is observed by the follow-up server message (0x9C/0x9D below) or a state read, never by an error reply.

Every item handler runs shared checks first (X inventory.md 5.1, addresses in game.exe):

| Check | Address | Passes when |
|---|---|---|
| cursor item | 005490e0 | item exists, mode 4 (on cursor), and is the player's cursor item |
| stored item | 00549150 | item exists, mode 0, in the player's inventory |
| stored or equipped | 005491b0 | item missing, or mode not 0/1, or in the player's inventory (inverted: a missing item passes) |
| owned item | 00549220 | in the player's inventory or the cursor item |
| belt item | 005492f0 | item missing, or mode not 2, or in the player's inventory |
| ground or owned | 00549350 | modes 0,1,2,4 as owned; mode 3 (ground): same act, within 10 subtiles per axis |
| item-move gate | 00535610 | no open interaction (and player data +0x4C = 0), or an NPC interaction that is not in the "talking" state |
| busy | 00535060 | player has an interaction, or a cursor item, or player data +0x4C != 0 |
| targeting reset | 0055bf50 | runs first in most handlers; clears item flag 0x4, sends 0x3F |

Item "mode" values used below (S X): 0 stored in a grid, 1 equipped, 2 in the belt, 3 on the ground, 4 on the cursor, 6 socketed inside another item. Inventory page ids in packets and ItemData (offset 0x45, D2MOO https://github.com/ThePhrozenKeep/D2MOO @ cbde180 source/D2Common/include/D2Inventory.h: INVPAGE 0 inventory, 1 equip, 2 trade, 3 cube, 4 stash, 5 belt, 255 none; S): 0x18 accepts page 0, 3, 4 (2 only while trading, 4 only in a town level, 1 and 5+ refused: X 7.3). ItemData bytes 0x68 and 0x69 (D2MOO names nNodePos / nNodePosOther; D2BS and our memory doc name them GameLocation / NodePage): 0x68 = grid index + 1 (0 when not in a grid), 0x69 = node page, 1 storage grid, 2 belt, 3 body slot, 4 body item with x >= 11 (S, D2MOO source/D2Common/src/D2Inventory.cpp lines ~1097-1128 and the NODEPAGE enum in D2Inventory.h; this answers part of Q8 in docs/d2-1.14d-memory.md, still to be probed live: U for 1.14d).

## Client to server item packets

Layout notation: field:type@offset. Handler address is the id's dispatcher entry; "-> inner" is the routine that does the work (X). Sizes are exact.

| Id | Name (X) | Size | Layout | Handler -> inner | Gate and failure (S, X inventory-moves.md section) |
|---|---|---|---|---|---|
| 0x16 | PickItem | 13 | type:u32@1 id:u32@5 cursor:u32@9 | 0054aad0 | 7.1: type > 5 -> 2; item must be mode 3 within 50 subtiles, else 1; farther than 5 subtiles or colliding: server WALKS the player to it and returns 0 (no pickup yet: send again on arrival); cursor != 0 puts it on the cursor, 0 auto-places (8.1) |
| 0x17 | DropItem | 5 | item:u32@1 | 0054ab40 | 7.2: needs cursor item; drops at the player's feet; result 0 even when no ground spot is found |
| 0x18 | InsertItemInBuffer | 17 | item:u32@1 x:u32@5 y:u32@9 page:u32@13 | 0054abb0 | 7.3: cursor item; page rules above; grid fit test at (x,y), failure -> 3, item stays on the cursor |
| 0x19 | RemoveItemFromBuffer | 5 | item:u32@1 | 0054acd0 | 7.4: item stored; page 1 -> 3; refused with a cursor item (sends the 0x5A "can't do that" message); an idle player may lift only page 0 items unless a stash/cube interaction is open |
| 0x1A | EquipItem | 9 | item:u32@1 bodyloc:u8@5 | 0054ad90 | 7.5: cursor item; bodyloc must be 1..10 else 2; requirement check (level/str/dex) refuses silently (sound, stat refresh) |
| 0x1B | Swap2HandedItem | 9 | item:u32@1 bodyloc:u8@5 | 0054ae30 -> 00563d20 | 7.6: bodyloc 4 or 5; other hand must hold an item; displaced item becomes the cursor item |
| 0x1C | RemoveBodyItem | 3 | bodyloc:u16@1 | 0054aec0 -> 00560cd0 | 7.7: no cursor item (0); empty location 0; location 8 (belt) needs the belt-removal rule |
| 0x1D | SwapCursorWithBody | 9 | item:u32@1 bodyloc:u8@5 | 0054af50 -> 00560f00 | 7.8: equipped item in mode 1; requirements of the cursor item |
| 0x1E | Swap1HWith2H | 9 | item:u32@1 bodyloc:u8@5 | 0054b030 -> 00561220 | 7.9: bodyloc 4 or 5; the other hand's item moves to inventory page 0 (original bug: if no room it stays detached) |
| 0x1F | SwapCursorBufferItem | 17 | cursor:u32@1 target:u32@5 x:u32@9 y:u32@13 | 0054b0f0 -> 00561b00 | 7.10: target page 1 or 2 -> 3; target taken BEFORE the cursor item's fit test, and nothing is restored if it fails (original quirk); no cube into a cube |
| 0x20 | UseGridItem | 13 | item:u32@1 x:u32@5 y:u32@9 | 0054b1e0 -> 0055e170 | 7.11 (V(O) handler): item in inventory; x,y within 50 subtiles; item mode 0, `useable`, no cursor item, not busy |
| 0x21 | StackItems | 9 | src:u32@1 dst:u32@5 | 0054b300 -> 0055e7c0 | 7.12: src != dst; same class only; merges up to max stack, overflow stays in src |
| 0x22 | UnstackItems | 5 | item:u32@1 | 0054b380 | 7.13: always result 3, does nothing (original quirk) |
| 0x23 | ItemToBelt | 9 | item:u32@1 slot:u32@5 | 0054b3e0 -> 0055e9b0 | 7.14: cursor item must be beltable; slot given by the client; mode becomes 2 |
| 0x24 | ItemFromBelt | 5 | item:u32@1 | 0054b450 -> 00562250 | 7.15: refused if a cursor item exists; belt compaction after |
| 0x25 | SwitchBeltItem | 9 | cursor:u32@1 belt:u32@5 | 0054b4e0 -> 0055eb30 | 7.16: no numboxes or similar-item check |
| 0x26 | UseBeltItem | 13 | item:u32@1 on_merc:u32@5 (+u32@9 unused) | 0054b560 -> 00562390 | 7.17: item mode 2, `useable`; on_merc needs an expansion game and item types 76, 80, 81 (hpot, apot, wpot); no hireling -> the potion is used on the player |
| 0x27 | UseItemAction | 9 | target:u32@1 used:u32@5 (see conflict below) | 0054b280 -> 00561ed0 | 7.18 (V(O) handler): identify scroll/tome on an item, also repair-style uses; both ids must resolve on the player; a tome needs stat 0x46 (quantity) >= 1 |
| 0x28 | SocketItem | 9 | socketable:u32@1 target:u32@5 | 0054b650 -> 00562660 | 7.19: filler on the cursor, target identified, socketed with a free socket; refused during player trade |
| 0x29 | ScrollToBook | 9 | scroll:u32@1 book:u32@5 | 0054b710 -> 0055ef20 | 7.20: scroll mode 3/4, book mode 0, same spell; a full book (max stack) -> 0 |
| 0x2A | ItemToCube | 9 | item:u32@1 cube:u32@5 | 0054b790 | cube.md: puts the cursor/ground item into the cube |
| 0x32 | BuyItem | 17 | npc:u32@1 item:u32@5 transaction:u32@9 client_price:u32@13 | 0054bac0 | vendors; P names the 4th field nCost |
| 0x33 | SellItem | 17 | npc:u32@1 item:u32@5 item_mode:u16@9 client_price:u32@13 | 0054bb20 | P: nMode word@9 then 6 padding bytes (X and P disagree on whether @13 is a price) |
| 0x34 | IdentifyWithNpc | 5 | npc:u32@1 | 0054bba0 | identifies ALL unidentified items for gold (Cain) |
| 0x35 | Repair | 17 | npc:u32@1 item:u32@5 + flags (see conflict) | 0054bb60 | P: bRepairOne u32@9 (1 = this item, 0 = all); X: unread u16@9, repair_flags u32@13 |
| 0x37 | IdentifyGamble | 5 | item:u32@1 | 0054bc30 | identify a gambled item |
| 0x4C | Transmogrify | 5 | item:u32@1 | 0054c760 | a misc.txt transmogrify use, NOT the cube |
| 0x4F | ClickButton | 7 | button:u16@1 p1:u16@3 p2:u16@5 | 0054c7c0 -> 00568060 | v = (p1 << 16) or p2 as one u32. Buttons: 0x12 close stash, 0x13 withdraw, 0x14 deposit (-> 00564d50), 0x17 close cube, 0x18 transmute |
| 0x50 | DropGold | 9 | unit:u32@1 amount:u32@5 | 0054c800 -> 00535510 | 7.22: unit must be the player; 0 <= amount <= carried gold and <= carry cap (clvl x 10000); up to 32 gold piles are made |
| 0x60 | SwapWeapons | 1 | - | 0054ce70 -> 005616a0 | weapon switch (expansion) |
| 0x61 | MercItem | 3 | bodyloc:u16@1 | 0054d430 | equip or take back a mercenary item; slot 0 puts the cursor item on the merc, 1..10 takes it |
| 0x63 | ItemToBeltShift | 5 | item:u32@1 | 0054d520 | shift-click into the belt; V(O) (390,000-case differential in docs/research/pr-click-draft.md) |

Reading the table: P (community) matches X field-for-field for every row except 0x33 and 0x35 above. libd2's size table matches X for every row. N gives the same layouts as names only.

## Stash gold, in numbers (S, X vendors-2.md section 10.2; consistent with our itemverbs design)

- Preconditions for 0x12-0x14: the player has an open interaction of type 2 whose unit is the stash object (class 0x10B, 267) and both the player's and the stash's rooms are in a town level; otherwise nothing happens and no message is sent. Without any interaction the server sends 0x77 code 0x0C (close trade).
- 0x13 withdraw v: v <= 0 or v > stash gold: nothing. Carried gold + v > carry cap (clvl x 10000): refused with the "impossible" sound (0x2C event 19), nothing moves. Otherwise gold += v, stash -= v.
- 0x14 deposit v: v <= 0 or v > carried gold: nothing. The stash cap is the constant 2,500,000 (not level dependent). If stash + v fits, the full amount moves; if not but the stash is below the cap, a PARTIAL deposit fills it; a full stash does nothing. No message or sound in any case; the player's stat update shows the result.
- v is built from the two u16 fields, so p1 >= 0x8000 makes v negative and nothing happens.
- Test vectors in X: withdraw 40,000 with carried 60,000, stash 100,000 gives carried 100,000 (cap), stash 60,000; deposit 30,000 with carried 30,000 and stash 2,490,000 gives carried 20,000, stash 2,500,000.

## Server to client item messages

| Id | Name | Layout (S: X server-messages.tsv, P gs2client.json) |
|---|---|---|
| 0x9C | ItemActionWorld | action:u8@1 size:u8@2 category:u8@3 item:u32@4 then the item bit stream (variable) |
| 0x9D | ItemActionOwned | action:u8@1 size:u8@2 category:u8@3 item:u32@4 owner_type:u8@8 owner:u32@9 then the item bit stream |
| 0x3E | UpdateItemStats | 34 bytes: size byte then a stat bit stream (quantity, durability changes) |
| 0x22 | item skill update | 12 bytes: unit_type:u8@1 unit:u32@3 skill:u16@7 quantity:u8@9 body:u8@11 (tome/scroll skill charges) |
| 0x42, 0x3F, 0x7D, 0x47/0x48 | cursor clear / item flag reset / broken-repaired / unit update | layouts in X inventory-moves.md section 11 |
| 0x0A | RemoveUnit | type:u8@1 guid:u32@2 (6 bytes): item left its room (picked up) |
| 0x5A | "can't do that" | 40 bytes starting 5A 0E 01 (sent when 0x19 is refused because a cursor item exists) |

Item action ids that arrive in 0x9C/0x9D, with the client packet that causes each (S X item-actions.tsv; D2MOO names in brackets): 0x01 ground to cursor (pickup); 0x04 put in container (0x18, pickup auto-place) [PutInContainer]; 0x05 remove from container [RemoveFromContainer]; 0x06 equip; 0x07 indirect swap of a body item (two-hander); 0x08 unequip; 0x09 swap body item; 0x0A add quantity (0x21 stack); 0x0D swap in container (0x1F); 0x0E put in belt; 0x0F remove from belt; 0x10 swap in belt; 0x11 auto-unequip; 0x12 to cursor; 0x15 update stats (socket fill); 0x16 and 0x14 unknown purpose; 0x17 weapon switch. The server sends them from a deferred per-tick pass over the owner's update list (X 6.1-6.2), so the 0x9C/0x9D for a move arrives at the end of the tick that handled it, not inside the handler. The item bit stream format is in X specs/items/bitstream.md; community parsers for it exist (below), usable as format references only.

## Parsers and format references (not rules)

- https://github.com/blacha/diablo2 @ 45c91b3 (MIT, last commit 2022): `packages/packets/src/parser.item.ts` and the item tracker example parse 0x9C/0x9D; a TypeScript sniffer, useful to see the field order. S.
- https://github.com/nokka/d2s, https://github.com/dschu012/d2s: save-file item parsers; the 0x9C/0x9D item bit stream is the same compact item format as .d2s items (S, not compared here).

## Conflicts and open questions

1. **0x27 field order.** X: target@1, used@5 (the item to identify first). P: nItemGUID@1, nScroolGUID@5 (same order). Our docs/probes/fid/fid-identify.md writes `{27, u32 scroll id, u32 item id}` (scroll first). Two outside sources agree with each other and disagree with us. The item-verbs design maps its command as `IdentifyWith{scroll,item}`; if a raw packet were ever built from that, the order matters. Resolve by reading 0054b280's two arguments in the disassembly (cheap, offline) before any 0x27 packet is generated; until then U.
2. **0x35 repair layout and 0x33 sell layout**: X and P differ (above). Needs a disassembly read of 0054bb60 and 0054bb20 or a captured packet. U.
3. **0x26 byte @9**: P calls it Unused, X lists no field there (size 13 either way). S.
4. Handler result 1 vs 2 vs 3 differ in the classifier only (1 and 2 enqueue the message); there is no refusal reply. The verbs' own refusal reasons (item-verbs.md section 5) are therefore planned client-side from the checks in the shared table; the server never tells why.
5. Packets are what a real client sends; the item-verbs design submits d2d's own server commands instead (docs/design/item-verbs.md section 3), so these layouts matter for (a) the differential test plan (compare the commands with what the original handler would do) and (b) any future raw-packet client. Gold: the d2d command is StashGold{amount,deposit}; the original is 0x4F with button 0x13/0x14.

## Use for the item-verbs design

- The pre-plan checklist for each verb is the "Gate and failure" column plus the shared checks table: copy the order (cursor check first, then range, then fit).
- Two verbs have quirks the original keeps and a conformance test should reproduce or avoid: 0x1F takes the target before testing the fit (an unfitting swap leaves the target on the cursor), and 0x22 never succeeds.
- Walk-then-act: 0x16 on a far item does not pick it up; it starts a walk. A verb that picks up must either wait for arrival and re-send, or use d2d's own pickup command.
- Gold deposit has a partial-fill result; the verb should report both balances after, as the design already says.

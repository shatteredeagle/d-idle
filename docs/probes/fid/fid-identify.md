# fid-identify: Deckard Cain, scrolls and tomes, what needs identifying, game.exe vs d2d vs the bot

> status: finished 2026-11-11 by agent decomp10 (Sonnet), offline. Result: **the Cain rule is already V read upstream and is repeated here with the server side re-read; the tome rule is new (V read); no number was run (the paths are packet handlers with unit and stat-list machinery, not built).** No PR candidate. Consumers: the identify chore and the Cain errand (TASK-298 item identify commands, TASK-307 identify2).

Tags: V read (disassembly read in this run), S (upstream proof: `re/quests-act1.md` lines 1044-1089, `re/npc-menu.md` 109-112, `re/net-packets.md` 784/804), D, I, U. No emulator number: tags are V read / S; nothing here may back a PR without a run.

## Questions

**Q1. What counts as unidentified? (S, V read of the count)** An item without flag 0x10. The Cain count FUN_0062a530 (client and server) counts items without the flag that are **stored on page 0 (inventory) or page 3 (cube), or equipped**; **the stash and the belt are not counted** (S, quests-act1.md 1051-1055). The menu entry "Identify Items" (0xfb4) exists only when that count is non-zero (S 1046). Which created items get the flag clear: the SME note says only magic, rare, set, unique (and crafted) need identifying (docs/sme/mechanics-items-drops.md:136, S); the creation code that sets or clears 0x10 was not read here (U).

**Q2. Cain. (S, server side V read this run: FUN_00578460 reached from packet 0x34 / FUN_0054bba0)** Client FUN_004b2020 sends `{0x34, npc guid}`. The server: the NPC must be the player's talk target and a Cain (class 0x109 camp, 0xf4..0xf6 Acts 2 to 4, 0x208 Act 5), `n` = FUN_0062a530 unidentified count; if quest 4 (Cain's rescue) bit 0 (done) and bit 1 (reward due) are both clear it takes `n x 100` gold **at once** (FUN_00576d90) and **identifies nothing if the gold is short**; then it identifies every counted item (FUN_00562590) and answers. So: **free from any act's Cain once the rescue quest is done, otherwise 100 gold per item, all or nothing, one request for the whole pack, cube and worn items. No per-visit limit exists** (nothing stored in the player). d2d identifies carried, worn and belt items (`rules::identify_all`, npc-menu.md); the belt part differs from game.exe (stash and belt not counted), harmless, D1 below.

**Q3. Scroll and tome. (V read, FUN_0054b280 then FUN_00561ed0; the packet 27 handler: `{27, u32 scroll id, u32 item id}`)** The handler needs both ids to resolve on the player and the item stored or equipped; FUN_00561ed0 branches on the used item class: for a tome (item type 0x12, FUN_0062b400 returns 0x12 at 0x562020) **stat 0x46 (quantity) drops by 1, not below 0** (0x56203a: `sub edi, 1` with a floor at 0, written back with FUN_00627260) and the client message 0x1019 is sent; a scroll item (`isc`) is consumed (the removal call sits further in the function, 0x562078 onward, not read: U). So **one charge or scroll per item identified, whichever tool**.

**Q4. Can a scroll identify an item in the stash? (U)** The handler's item lookup is on the player's own lists (FUN_00549220); whether a stash item resolves is not read. The Cain count excludes the stash (Q1), so move items to the pack first.

## Deviations d2d vs game.exe

| id | Rule | Tag | Differential |
|---|---|---|---|
| D1 | d2d identifies belt items too (`identify_all`), game.exe's count stops at inventory, cube and worn | S, V read vs V read | buildable: FUN_0062a530 with a unit that has one unidentified item in each container; d2d `rules::unidentified` |
| D2 | d2d's price for the missing gold case (all-or-nothing, 100 per item) not read in d2d | U | read `rules::identify_all` callers; recipe below |

## Bot findings (not changed)

- B1 `internal/town/chores.go:155-166` `needIdentify` scans `ContInventory` only. Cain would also identify the cube and worn items (Q1/Q2), and one request covers all; for the Cain errand (TASK-298) the cheap path is one request when the rescue is done (free) or when `100 x n` is affordable (all or nothing: with less gold than `100 x n`, remove items from the pack first or split by value).
- B2 `internal/town/identify.go:33-58` identifies one item at a time with a tool: one scroll or one tome charge per item (Q3). The scroll purchase `buyIDScrolls` counts `len(todo)`: correct by Q3.
- B3 `internal/town/acts.go:37` lists Cain's npc ids per act (146, 244, 245, 246, 265, 520); camp Cain 265 is present before the rescue, so the 100-per-item price applies there (Q2). Nothing in the town code reads the quest bit; the price is therefore unknown to the planner: expose the quest 4 bit to `idScrollNeed` decisions (U whether the snapshot has it).
- B4 the bot's check "an item is a likely valuable" (`internal/town/basicsfix.go:15`) saves scrolls; at 100 gold per item Cain is dearer than a 25-gold scroll before the rescue; after it Cain is free, so there the knob should say "use Cain".

## Drill recipe (fork verbs, not run)

R1: lab d2d, `debug unid` (unidentify every carried item, `docs/forkcheats.md`), visit camp Cain before the rescue: expect the entry "Identify Items: N x 100" and nothing identified at gold below 100 N. R2: put one unidentified item on the belt (hp slot cannot hold it, so use a charm... skip: item must be 1x1 belt-legal: not possible); D1 stays a read. R3: tome of identify with `charges` seen in `dump items` before and after one identify: -1.

## Matrix proposal

IT07 identify: "Cain rule S/V read (free after the rescue, 100 per item before, all or nothing, pack+cube+worn); tome -1 charge per item V read; unid creation rule U; flow on d2d unverified live (Cain not reachable before the rescue)". 
